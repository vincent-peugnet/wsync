Wsync
=====

[Website](https://w.club1.fr/wcms/)

Is a cli synchronization client for the [W](https://w.club1.fr) wiki engine.

Usage is inspired by Git, combined with an interactive mode.

`api/client.go` is an independant package that take care of communicating with [W's API](https://github.com/vincent-peugnet/wcms/blob/master/API.md).

> ⚠️ For now, it's only supposed to work with pages V2


Quick start
-----------

First, you have to chose a folder on your computer that will be your local repository.

Then, open your terminal here.
The easyest way is to display the menu by typing:

    wsync

To initialize a repo, select "init" (which correspond to the [`init` sub-command](#init)).

A short form will ask you the **url of a W**, then your **username** and **password**.

To add the first pages, you can select "list" option in the menu.

Or you can directly list the ID of pages you want to sync using [`add` sub-command](#add):

    wsync add PAGE_1 PAGE_2 ...


Storage
-------

Each page content is stored as a mardown file.
The name of the file match the ID of the page followed by the `.md` extension.

A hidden `.wsync` folder also live in this repo.
It contains the token used to authenticate and keep track of sync dates.


Synopsis
--------

    wsync [-C PATH] [-F] | init [W_URL]
                         | status
                         | [-i] sync [PAGE_ID...]
                         | push [PAGE_ID...]
                         | pull [PAGE_ID...]
                         | remove PAGE_ID...
                         | add PAGE_ID...
                         | list
                         | version

### Flags

- `-C PATH` Run as if wsync was started in `PATH` instead of the current working directory.
- `-F` Force `push` and `pull` sub-commands in case of conflict.
- `-i` interactive mode. Allow to choose a version in case of conflict.


### Sub-commands

#### init

    wsync init

If the directory is empty and writtable, this will initialize a local repo.
The command is interactive and will ask you for the server URL, username and password.

Alternatively, the server URL can be indicated as the first argument. Example:

    wsync init https://mywiki.com


#### status

    wsync status

Print the current status of local pages.


#### sync

    wsync [-i] sync

This will bi-directonnaly synchronise the pages:

- Localy edited pages will be pushed.
- Remotely edited pages will be pulled.

If both side where edited, a conflict is triggered.

If interactive mode is on (flag `-i`), each conflict let you choose which version to keep (local or server).

> 💡 Using the menu will automatically enable interactive mode.


#### push

    wsync [-F] push [PAGE_ID...]

Will push to the server all edited pages. If force option is activated (flag `-F`), conflict will be resolved by erasing the server version with the local one.

If page IDs are provided as arguments, only listed pages will be pushed.


#### pull

    wsync [-F] pull [PAGE_ID...]

Will pull to the server all edited pages. If force option is activated (flag `-F`), conflict will be resolved by erasing the local version with the server one.

If page IDs are provided as arguments, only listed pages will be pulled.


#### remove

    wsync remove PAGE_ID...

For each provided page ID:

- If the local version of page is the same as the server, the local file is deleted.
- If the version is different, the local file is kept, but is un-tracked.
Un-tracked files are not concerned by `sync`, `push` or `pull`.


#### add

    wsync add PAGE_ID...

For each provided page ID:

If the page exist on the server, is un-tracked, and no file correspond locally,
a new file is created in the repo and added to the list of tracked pages.
Otherwise, an error message is printed.


#### list

    wsync list

A interactive list of all pages on the server is displayed. You can check or un-check pages in order to **add** or **remove** them from the tracked pages.


#### version

    wsync version

Output the current software version.


Installation
============

Build
-----

Build software for your machine.

    make

Will create one file:

    wsync

It should be copied to a folder part of your PATH.


Publish a new release
---------------------

The release process uses GitHub's CLI so you will need to have it installed and authenticated
(`sudo apt install gh`).

Then, to make the release, run one of the following command:

    make release-patch
    make release-minor
    make release-major


### multi OS/Arch build

Build software for linux (arm/amd), mac (arm/amd) and windows.

    make all

Will create 5 files:

```
wsync-linux-amd64
wsync-linux-arm64
wsync-macos-amd64
wsync-macos-arm64
wsync-windows-amd64.exe
```

Other
-----

To delete all generated files, run:

    make clean
