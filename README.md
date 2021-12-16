# Switch Host

Yet another local switch game backup management tool.

This is designed to be left running in the background as a small, low resource service.
As such this is probably not going to serve thousands of people (It _could_ maybe, but untested).

This is very much a WiP at the moment and nothing is suuper locked in.
That said, I have no _intent_ of breaking anything that exists now.
Most likely changes are to do with file pathing in served files and cleaning up.

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
1. Setup watch of incoming folder

## Keys (optional)

Having a prod.keys file will allow you to ensure the files you have a correctly classified. The app will look for the `prod.keys` file in `${HOME}/.switch/`
If keys are missing some features (sorting) will not function as of present
Note: Only the header_key, and the key_area_key_application_XX keys are required.

## Structure

The code is split into a bunch of sperate packages to keep the code ever so slightly re-usable.
There is still a bunch of interdependencies to be cleaned up as time permits.
