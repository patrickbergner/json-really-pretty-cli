jrp - JSON Really Pretty

Usage:
  jrp [flags] [file ...]

Reads from stdin when given no files or "-", and writes to stdout unless
-w/--write is set. Objects and arrays are collapsed onto one line unless they
hold an object somewhere inside them, so flat records stay compact and
structured data opens up. -x/--max-width and -n/--max-items add hard caps
that fall back to conventional one-member-per-line output.

Flags:
  -w, --write             rewrite each named file in place instead of writing to stdout
  -i, --indent int        number of spaces per indentation level (default 4)
  -t, --tab               indent with tabs instead of spaces
  -x, --max-width int     expand containers whose one-line form would pass this column (0 disables, default 0)
  -n, --max-items int     expand containers with more than this many members or elements (0 disables, default 0)
  -b, --tight-braces      write inline objects as {"a": 1} instead of { "a": 1 }
  -p, --padded-brackets   write inline arrays as [ 1, 2 ] instead of [1, 2]
  -s, --sort-keys         sort object keys instead of keeping the input order
  -v, --version           print the version and exit
  -h, --help              show this help message
