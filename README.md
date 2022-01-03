# Switch Host

Yet another local switch game backup management tool.

Have a messy collection of backups you have taken? This is a solution for you.

Switch Host will parse the metadata in each file, validate each files integrity and sort them for you.
Additionally, it includes a server to allow easier transfer of files. This serves over both HTTP and FTP (with auth).

Most features can be turned on and off at will and it should do its best.
Having a keys file from your switch is _REQUIRED_ for functionality.

The designed behaviour is to have a folder for the "library" to to be stored in

Along with sorting and serving the files, this can provide a _shop_ file and it serves a fairly minimal web-browser listing.

Extended validation of file hashes ensures that the files you have backed up are intact copies, and allows you to detect bitrot.
Scanning can be performed on all files in the library, incoming files or both.
It is reccomended to scan incoming files to ensure that your backups are intact before storing.

## Features

1. Scans multiple folders for source files
1. Organise files into one unified structure
1. -> Cleans up empty folders after files are moved
1. Validate SHA256 checksums of file contents before moving to library and storing
1. Fairly nice text user interface to see the status of the system
1. Optionally compress files via NSZ (external tool)
1. Supports TitleDB or reading file metadata for names (both by default)
1. Serves files over FTP and HTTP, and supports generating a `json` shop index
1. -> Actual filenames are hidden, and virtual file paths are used when serving
1. Minimal webUI shows tiles of all tracked backups
1. Seamless settings file updates
1. Does **NOT** use a database of any form, just keeps things in ram (pro: cant break state and con: has to scan files at start)
1. Can run easily on a Raspberry Pi

## Running the program from source

1. Compile program via `go build`
1. Run program `./switchhost` in a terminal

## First run (How To)

When the program is first run, a `config.json` file will be created with the safe defaults (most features turned off).
Press control-c to stop the program running and then open the configuration file to edit it.

### Configuration file

The configuration file will be generated at first run with safe defaults. You **will** want to change these for proper use.

At the least you will want to set `sourceFolders`, `storageFolder`, `users`.
Set `storageFolder` to the file path that you want the collection stored in.
`sourceFolders` are locations for the software to scan on boot.

After this, you can run the software again and check the log to see that files are imported found correctly.

I reccomend running once with `validateLibrary` turned on to check all of the existing files are intact.

Turning on `enableSorting` will enable moving the files from `sourceFolders` into the `storageFolder` and also rename files to match the naming scheme set in `organisationFormat`.

Most options should be fairly straightforward as to what they turn on/off.

## Keys (required)

Having a prod.keys file will allow you to ensure the files you have a correctly classified. The app will look for the `prod.keys` file in `${HOME}/.switch/` and in the program folder.
If keys are missing some features (sorting) will not function as of present
Note: Only the header_key, and the key_area_key_application_XX keys are required; if you dont have these you will need to dump them from your switch.

## Architecture

On startup a _bunch_ of workers are started. These are used to perform various actions during library management. You can view their status in the terminal UI of the application.

When a file is "scanned" into the library, the following chain of events occurs

`Scanner` -> `Metadata parser` -> `Validator` -> `Organiser` -> `Cleanup` -> `Compression`

### Scanner

Scans a list of folders plus the library at startup, and queues all found files for metadata parsing.

### Metadata parser

This will read the headers from the file in order to figure out the titleID, version number, and file type.

### Validator

If validation is turned on for the file coming in, the data portions of the file are read and the SHA256 checksums are checked against those stored in the headers. If the checksum matches the file is sent onwards to be orgnaised. If the checksum fails an warning is logged to the log, and optionally the file is deleted.

### Organiser

This uses the parsed TitleID information to generate the organised file path and moves the file there (if its not already there).

Once the file has been moved its added to the in-memory index and is available for serving on the server.

If enabled, the file will be sent to be compressed if is not already.

### Cleanup

This task is notified when files are deleted and will cleanup the containing folder from having empty folders hanging around

### Compression

This calls out to the nsz program if enabled to compress the NSP/XCI into its compressed form. If compression works, the old file is dropped from the library and the new file will be added in its place.

## Further work (coming)

1. Add ability to compress files to NSZ/XCZ using pure golang
1. More detailed web-ui
1. Empty folder cleanup needs rework, its rather messy at the moment
