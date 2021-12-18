# Switch Host

Yet another local switch game backup management tool.

This is designed to be left running in the background as a small, low resource service.
As such this is probably not going to serve thousands of people (It _could_ maybe, but untested).

The goal is to provide a reasonable all in one management tooling to keep things organised and serve files across your LAN.

## Features

1. Scans multiple folders for source files
1. Optionally organise based on user specified pattern
1. -> Cleans up empty folders after files are moved
1. Supports TitleDB or reading files for names (both by default)
1. Serves files over FTP and HTTP, and supports generating a `json` index
1. Super minimal (so far) WebUI baked in
1. Actual filenames are hidden, and virtual file paths are used when serving
1. Seamless settings file updates
1. Does **NOT** use a database of any form, just keeps things in ram (pro: cant break state and con: has to scan files at start)
1. All built in; no external dependencies other than running `go build`

## Running the program

1. Compile program via `go build`
1. Run program `./switchhost`

After running the first time a `config.json` will be created with the default values, these can be modified as desired

## Further work (coming)

1. Authentication
1. More detailed web-ui
1. Add ability to compress files to NSZ/XCZ
1. Empty folder cleanup needs rework, its rather messy at the moment

## Keys (semi-optional)

Having a prod.keys file will allow you to ensure the files you have a correctly classified. The app will look for the `prod.keys` file in `${HOME}/.switch/` and in the program folder.
If keys are missing some features (sorting) will not function as of present
Note: Only the header_key, and the key_area_key_application_XX keys are required; if you dont have these you will need to dump them from your switch.

## Structure

The code is split into a bunch of sperate packages to keep the code ever so slightly re-usable.
There is still a bunch of interdependencies to be cleaned up as time permits.
