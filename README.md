# WatchFiles

Will watch change on files (modification time/size) in a polling fashion and restart the command if needed.

Meant to be used in docker container where inotify is not working for shared folders to releoad a foreground process.

# Usage

```
Usage of ./watchFiles:
  -a string
        the args to pass to the command.
  -c string
        The command to run.
  -f value
        List of files to watch.
```

Example:

`watchFiles -f file1 -f file2 -c python -a -m -a SimpleHTTPServer -a 4000`

Note: Your command must stay in foreground to this to work properly.
