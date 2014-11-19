### Glyph import dependencies visualiser

To install: `go get github.com/davecheney/junk/glyph/...`  
To use, run `./glyph` to start the webserver that allows you to visualise import paths.

The URLs take the form `http://localhost:8080/<visual_name>/package/import/path`. The following visualizations are available: `radial`, `cluster`, `force`, `forcegraph`, `forceimports` and `chord`.

Additionally, tangible data can be obtained in various formats by using `data`, `links`, `imports`, `csv`, `csvimports` in place of the visual names.

Example URL (considering you have the "profile" package installed):  
`http://localhost:8080/chord/pkg/profile`
