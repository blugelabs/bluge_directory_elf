# bluge_directory_elf

An implementation of the Bluge Directory interface which can operate on index data which has been stored inside additional sections of an elf executable.

This can be useful to create an executable with search code, that can access index data attached to the executable itself.

### Usage

To begin, you need an existing Bluge index.  While not required, it is recommended that one builds an index with the Bluge Offline Index Writer.  This creates an index with single segment, which is optimized for this use case.

Next, create an application which has some search logic.  This could be a command-line program which takes search queries from the command-line arguemnts, or it could be a serverless executable desgined to service HTTP requests.  This program should use the provided Directory implementation to access the index in a read-only manner:

```
	cfg := bluge.DefaultConfigWithDirectory(func() index.Directory {
		return bluge_directory_elf.NewElfDirectory(os.Args[0], "index")
	})

	reader, err := bluge.OpenReader(cfg)
	if err != nil {
		log.Fatalf("error opening index reader: %v", err)
	}
```

In this example we see that the application creates a Bluge configuration using this module's Elf Directory implementation, and it passes itself (`os.Args[0]`) as the elf-executable.
The second argument here is a hard-coded index named `index`.  This allows you to store and reference multiple indexes in the executable.
Finally, one opens a reader using this configuration.  From this point it behaves like any other index reader.

Next one compiles this application.

The last step is to combine these two pieces.  We want to take the index we built, and add it to the elf-executable we just compiled.  This can be done with the provided application.

**NOTE**: this requires the host machine have the `objcopy` application installed.

The provided `bluge_add_to_elf` command takes 3 arguments:

- path to elf-executable to modify
- name of the index (to allow for multiple indexes in the executable)
- path to the exiting bluge index

For example:

```
$ bluge_add_to_elf compiled-application index path-to-index
```