# Wsync

Is a cli synchronization client for the [W](https://w.club1.fr) wiki engine.

Usage is inspired by Git, combined with an interactive mode.

`api/client.go` is an independant package that take care of communicating with [W's API](https://github.com/vincent-peugnet/wcms/blob/master/API.md).


## Usage

First, you have to chose a folder on your computer that will be your local repo.

Then, open your terminal here.
The easyest way is to use the interactive mode by typing:

    wsync

A menu will open (that match the sub-commands).
To initialize a repo, select "init".

A short form will ask you the url of a W, then your username and password.

To add the first pages, you can select "list" option in the menu.

## Synopsis

    wsync [-C PATH] [-F] | init [W_URL]
                         | status
                         | sync [...PAGE_ID]
                         | push [...PAGE_ID]
                         | pull [...PAGE_ID]
                         | remove ...PAGE_ID
                         | add ...PAGE_ID
                         | list

### Flags

- `-C PATH` Run as if wsync was started in `PATH` instead of the current working directory.
- `-F` Will force `push` and `pull` sub-commands in case of conflict.
