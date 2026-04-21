# Benchmark

## Dependencies

- [unzip](https://man.archlinux.org/man/unzip.1)
- [Qlever](https://docs.qlever.dev/quickstart/#installing-qlever)

```sqarl
CONSTRUCT { ?s  ?p ?o }  WHERE  { 
      ?s  ?p ?o. 
     FILTER (?p != <http://vocab.sindice.net/analytics#cardinality>)
	 } LIMIT 100
```
ls engine/**/*.{nt,n3,owl,rdf}
