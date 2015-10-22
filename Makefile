# add scuttlebutt to .ssh/config
deploy:
	GOOS=linux go build -o /tmp/scuttlebuttd ./cmd/scuttlebuttd
	gzip -f /tmp/scuttlebuttd
	scp /tmp/scuttlebuttd.gz scuttlebutt:/usr/local/bin/scuttlebuttd.gz
	ssh -T scuttlebutt "service scuttlebuttd stop; gunzip -f /usr/local/bin/scuttlebuttd.gz && service scuttlebuttd start"
