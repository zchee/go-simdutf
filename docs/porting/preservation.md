# Phase 0 user-asset preservation

`preservation-manifest.tsv` is the Phase 0 baseline for pre-existing user-owned
assets and runtime state. It is deliberately stored outside every audited
runtime tree.

The TSV columns are:

```text
path class type mode size sha256 symlink_target baseline_timestamp_utc notes
```

`immutable` covers `.agents/**`, `.claude/**`, `AGENTS.md`, and `CLAUDE.md`.
Every immutable entry must retain its path, lstat type, mode, lstat size, and
SHA-256. A regular-file digest covers file bytes. A symlink digest covers the
link-target bytes (without a newline), and `symlink_target` records that target
literally.

`mutable` covers `.omx/**` and `.omc/**`. Baseline paths must retain their lstat
type and mode, but directory size, regular-file size/content, symlink target,
and digest may change and are reported rather than rejected. New paths below a
mutable root are reported as additions. Deletion, type changes, and mode changes
of baseline mutable paths fail the preservation gate. If `.omc` is absent when
the baseline is generated, it is represented by one `.omc` row with type
`absent`; later creation is then reported as a mutable addition.

Directories have no portable byte stream to hash. Their `sha256` and
`symlink_target` fields are therefore empty; their lstat size is recorded, and
their children are represented as individually sorted rows. Immutable directory
size is compared exactly. Mutable directory size is audit-only because runtime
children may be added or removed.

## Generate the baseline

Run from the repository root. This command does not write below any audited
path. The temporary file is outside the audited roots, and the output manifest
itself is explicitly excluded from traversal.

```sh
baseline_timestamp_utc=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
preservation_tmp=$(mktemp "${TMPDIR:-/tmp}/simdutf-preservation.XXXXXX")
printf 'path\tclass\ttype\tmode\tsize\tsha256\tsymlink_target\tbaseline_timestamp_utc\tnotes\n' >"$preservation_tmp"
BASELINE_TIMESTAMP_UTC="$baseline_timestamp_utc" perl -MDigest::SHA=sha256_hex -MFile::Find -MFcntl=:mode <<'PERL' >>"$preservation_tmp"
use strict;
use warnings;

my $timestamp = $ENV{BASELINE_TIMESTAMP_UTC};
my $manifest = 'docs/porting/preservation-manifest.tsv';
my @roots = qw(.agents .claude AGENTS.md CLAUDE.md .omx);
push @roots, '.omc' if -e '.omc' || -l '.omc';
my @paths;
find({ no_chdir => 1, wanted => sub {
    (my $path = $File::Find::name) =~ s{^\./}{};
    push @paths, $path unless $path eq $manifest;
}}, @roots);

for my $path (sort @paths) {
    my $class = $path =~ m{^(?:\.agents(?:/|$)|\.claude(?:/|$)|AGENTS\.md$|CLAUDE\.md$)}
        ? 'immutable'
        : $path =~ m{^(?:\.omx|\.omc)(?:/|$)} ? 'mutable' : die "unclassified: $path\n";
    my @st = lstat $path or die "lstat $path: $!\n";
    my $type = S_ISLNK($st[2]) ? 'symlink'
        : S_ISDIR($st[2]) ? 'directory'
        : S_ISREG($st[2]) ? 'regular' : die "unsupported type: $path\n";
    my $mode = sprintf '%04o', S_IMODE($st[2]);
    my ($digest, $target) = ('', '');
    if ($type eq 'regular') {
        open my $fh, '<', $path or die "open $path: $!\n";
        binmode $fh;
        $digest = Digest::SHA->new(256)->addfile($fh)->hexdigest;
    } elsif ($type eq 'symlink') {
        $target = readlink $path;
        $digest = sha256_hex($target);
    }
    my $notes = $class eq 'immutable'
        ? 'pre-existing user asset; immutable byte equality required'
        : 'runtime state; content and size may change; preserve baseline path/type/mode';
    print join("\t", $path, $class, $type, $mode, $st[7], $digest,
        $target, $timestamp, $notes), "\n";
}

if (!-e '.omc' && !-l '.omc') {
    print join("\t", '.omc', 'mutable', 'absent', '', '', '', '', $timestamp,
        'runtime state absent at baseline; later creation is reported'), "\n";
}
PERL
LC_ALL=C sort -t "$(printf '\t')" -k1,1 "$preservation_tmp" -o "$preservation_tmp"
sed -i.bak '/^path\tclass\ttype\tmode\tsize\tsha256\tsymlink_target\tbaseline_timestamp_utc\tnotes$/d' "$preservation_tmp"
rm -f "$preservation_tmp.bak"
{ printf 'path\tclass\ttype\tmode\tsize\tsha256\tsymlink_target\tbaseline_timestamp_utc\tnotes\n'; cat "$preservation_tmp"; } >docs/porting/preservation-manifest.tsv
rm -f "$preservation_tmp"
```

## Recheck the preservation gate

Run from the repository root. The check is read-only. It validates the schema,
unique paths, classifications, digest formats, complete immutable equality, and
the mutable path/type/mode contract. It also reports mutable content/size
changes and additions. A zero exit status means the preservation gate passes.

```sh
perl -MDigest::SHA=sha256_hex -MFile::Find -MFcntl=:mode <<'PERL' || exit 1
use strict;
use warnings;

my $manifest = 'docs/porting/preservation-manifest.tsv';
open my $fh, '<', $manifest or die "open $manifest: $!\n";
chomp(my $header = <$fh> // '');
my $expected_header = join "\t", qw(path class type mode size sha256 symlink_target baseline_timestamp_utc notes);
die "invalid header\n" unless $header eq $expected_header;

my (%rows, %counts, @failures, @mutable_content_changes, @mutable_directory_size_changes);
my $line = 1;
while (<$fh>) {
    ++$line;
    chomp;
    my @f = split /\t/, $_, -1;
    push @failures, "line $line: expected 9 columns" and next unless @f == 9;
    my ($path, $class, $type, $mode, $size, $digest, $target, $timestamp) = @f;
    my $expected_class = $path =~ m{^(?:\.agents(?:/|$)|\.claude(?:/|$)|AGENTS\.md$|CLAUDE\.md$)}
        ? 'immutable' : $path =~ m{^(?:\.omx|\.omc)(?:/|$)} ? 'mutable' : '';
    push @failures, "line $line: duplicate path $path" if exists $rows{$path};
    push @failures, "line $line: invalid class $class" unless $class eq 'immutable' || $class eq 'mutable';
    push @failures, "line $line: unclassified path $path" unless length $expected_class;
    push @failures, "line $line: class mismatch for $path ($class != $expected_class)"
        if length($expected_class) && $class ne $expected_class;
    push @failures, "line $line: invalid type $type" unless $type =~ /\A(?:regular|directory|symlink|absent)\z/;
    push @failures, "line $line: absent type is valid only for mutable .omc" if $type eq 'absent' && !($class eq 'mutable' && $path eq '.omc');
    push @failures, "line $line: invalid mode $mode" unless $type eq 'absent' ? $mode eq '' : $mode =~ /\A0[0-7]{3}\z/;
    push @failures, "line $line: invalid size $size" unless $type eq 'absent' ? $size eq '' : $size =~ /\A\d+\z/;
    push @failures, "line $line: invalid digest for $type" unless
        $type eq 'regular' || $type eq 'symlink' ? $digest =~ /\A[0-9a-f]{64}\z/ : $digest eq '';
    push @failures, "line $line: invalid symlink target field" unless $type eq 'symlink' ? length($target) : $target eq '';
    push @failures, "line $line: invalid UTC timestamp" unless $timestamp =~ /\A\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z\z/;
    $rows{$path} = \@f;
    ++$counts{$class};
}

sub current_entry {
    my ($path) = @_;
    my @st = lstat $path or return;
    my $type = S_ISLNK($st[2]) ? 'symlink'
        : S_ISDIR($st[2]) ? 'directory'
        : S_ISREG($st[2]) ? 'regular' : 'other';
    my ($digest, $target) = ('', '');
    if ($type eq 'regular') {
        open my $in, '<', $path or die "open $path: $!\n";
        binmode $in;
        $digest = Digest::SHA->new(256)->addfile($in)->hexdigest;
    } elsif ($type eq 'symlink') {
        $target = readlink $path;
        $digest = sha256_hex($target);
    }
    return ($type, sprintf('%04o', S_IMODE($st[2])), $st[7], $digest, $target);
}

for my $path (sort keys %rows) {
    my ($base_path, $class, $type, $mode, $size, $digest, $target) = @{$rows{$path}};
    my @now = current_entry($path);
    if ($type eq 'absent') {
        next unless @now;
        next; # creation is counted below as a mutable addition
    }
    if (!@now) {
        push @failures, "$class baseline path missing: $path";
        next;
    }
    push @failures, "$class type changed: $path ($type -> $now[0])" if $now[0] ne $type;
    push @failures, "$class mode changed: $path ($mode -> $now[1])" if $now[1] ne $mode;
    if ($class eq 'immutable') {
        push @failures, "immutable size changed: $path ($size -> $now[2])" if $now[2] ne $size;
        push @failures, "immutable digest changed: $path" if $now[3] ne $digest;
        push @failures, "immutable symlink target changed: $path" if $now[4] ne $target;
    } elsif ($type eq 'directory') {
        push @mutable_directory_size_changes, $path if $now[2] ne $size;
    } else {
        push @mutable_content_changes, $path
            if $now[2] ne $size || $now[3] ne $digest || $now[4] ne $target;
    }
}

my @current_mutable;
for my $root (grep { -e $_ || -l $_ } qw(.omx .omc)) {
    find({ no_chdir => 1, wanted => sub {
        (my $path = $File::Find::name) =~ s{^\./}{};
        push @current_mutable, $path unless $path eq $manifest;
    }}, $root);
}
my @added = sort grep { !exists $rows{$_} || $rows{$_}[2] eq 'absent' } @current_mutable;

printf "rows=%d immutable=%d mutable=%d mutable_content_changes=%d mutable_directory_size_changes=%d mutable_additions=%d failures=%d\n",
    scalar(keys %rows), $counts{immutable} // 0, $counts{mutable} // 0,
    scalar(@mutable_content_changes), scalar(@mutable_directory_size_changes), scalar(@added), scalar(@failures);
print "mutable content/size change: $_\n" for @mutable_content_changes;
print "mutable directory-size change: $_\n" for @mutable_directory_size_changes;
print "mutable addition: $_\n" for @added;
print STDERR "FAIL: $_\n" for @failures;
exit(@failures ? 1 : 0);
PERL
shasum -a 256 docs/porting/preservation-manifest.tsv
```

At final signoff, retain the command output with the manifest SHA-256. Any
immutable difference or mutable baseline deletion/type/mode change is a blocker;
mutable byte/size changes and additions are evidence to review for expected
runtime use, not silent preservation failures.
