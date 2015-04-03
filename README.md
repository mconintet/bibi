## bibi

This is a scanner, it can scan all available resources within some website according the dictionary you specified.

## Usage

```shell
Usage:
  -c=5: concurrence count
  -d="": dictionary
  -h="": host
  -l="log.txt": file to save log
  -t=30: timeout seconds for per request
```

```
go run bibi.go -h "http://somesite.com" -d "/path/to/your/dict.txt"
```

## Example

```shell
$go run bibi.go -h "http://www.iana.org/" -d "dict2.txt"
Calculating count of lines in dictionary...
Done, lines count is: 1

Detecting, concurrent count is [5] ...
|====================================================================================| 100.00%
All:  1
Succeed:  1
Failed:  0
Errors:  0
Matches:  [http://www.iana.org/domains/reserved]
```
