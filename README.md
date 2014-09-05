# spot-log

Poll the SPOT Satellite Tracking System API in regular intervals for the position of a tracker, send the recorded positions to stdout.

This is pretty much alpha. Call it with a “shared feed ID” as its first (and only) command-line parameter:

	go get github.com/scy/spot-log
	spot-log 8dJKed8Sjlkfdj89jsDH89EHDl

It will start reading the feed backwards until all previous values have been retrieved. Then, it will poll the API for new values.

All values that are retrieved are printed instantly to stdout. Additionally, for debugging purposes the URL that’s being queried is shown as well.

**The positions will not be printed in chronological order.**

The tool will wait for 3 minutes between API calls in order to be friendly to the SPOT servers.
