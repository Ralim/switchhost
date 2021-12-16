# Switch Host

Yet another local switch game backup management tool.
This is designed to be left running in the background as a small, low resource service.

## Features

1. Scans multiple folders for source files
1. Optionally organise based on user specified pattern
1. -> Cleans up empty folders after files are moved
1. Supports TitleDB or reading files for names
1. Serves files over FTP and HTTP, and supports generating a `json` index
1. Super minimal (so far) WebUI baked in
1. Actual filenames are hidden, and virtual file paths are used when serving
1. Seamless settings file updates
1. Does **NOT** use a database of any form, just keeps things in ram (pro and con)

## Further work (coming)

1. More detailed web-ui
1. Add ability to compress files to NSZ/XCZ
1. Empty folder cleanup needs rework, its rather messy at the moment

## Keys (optional)

Having a prod.keys file will allow you to ensure the files you have a correctly classified. The app will look for the `prod.keys` file in `${HOME}/.switch/`
If keys are missing some features (sorting) will not function as of present
Note: Only the header_key, and the key_area_key_application_XX keys are required.
