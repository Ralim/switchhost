# Switch Host

Yet another local switch game backup management tool.
This is designed to be left running in the background as a small, low resource service

## Features

1. Scans multiple folders for source files
1. Optionally organise based on user specified pattern
1. Supports TitleDB or reading files for names
1. Serves files over FTP and HTTP, and supports generating a `json` index

## Goals (Todo)

1. Serves a minimal web-ui for administration
1. Can automate compression of files

## Keys (optional)

Having a prod.keys file will allow you to ensure the files you have a correctly classified. The app will look for the `prod.keys` file in `${HOME}/.switch/`

Note: Only the header_key, and the key_area_key_application_XX keys are required.
