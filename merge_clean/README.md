# merge-clean-nt

Merges and cleans [Large RDF Bench](https://github.com/dice-group/LargeRDFBench) N-Triples datasets by percent-encoding invalid characters inside IRI tokens.

## Dependencies

- [Go 1.21+](https://go.dev/)
- [GNU Make](https://www.gnu.org/software/make/)

## Usage

Build the binary:

```zsh
make build
```

The binary will be built at `./bin/merge_clean_nt`.

```
Usage of ./bin/merge_clean_nt:
  -o string
        output file (default: stdout)
```

Merge and clean a glob of N-Triples files:

```zsh
./bin/merge_clean_nt -o merged_clean.nt 'engine/Affymetrix/*.nt'
```

Stream to stdout:

```zsh
./bin/merge_clean_nt 'engine/Affymetrix/*.nt' > merged_clean.nt
```

Multiple globs:

```zsh
./bin/merge_clean_nt -o merged_clean.nt 'engine/Affymetrix/*.nt' 'engine/DBPedia-Subset/*.nt'
```

Stdin fallback (no glob args):

```zsh
cat merged.nt | ./bin/merge_clean_nt > merged_clean.nt
```

## Encoded characters

The following characters are percent-encoded when found inside an IRI (`<...>`):

| Character | Encoded |
|-----------|---------|
| space     | `%20`   |
| `"`       | `%22`   |
| `^`       | `%5E`   |
| `{`       | `%7B`   |
| `\|`      | `%7C`   |
| `}`       | `%7D`   |
| `\`       | `%5C`   |
| `` ` ``   | `%60`   |

Characters outside of IRI tokens are passed through unchanged.

## Development

Run tests:

```zsh
make test
```

Run linter:

```zsh
make lint
```
