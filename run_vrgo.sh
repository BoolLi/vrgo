#!/bin/bash
primary_cmd="$GOPATH/bin/vrgo --mode=server --port=1234 --id=1 --backup_ports=9000 --backup_ports=9001 --backup_ports=9002> $GOPATH/bin/primary.log&"

backup_cmd="$GOPATH/bin/vrgo --mode=backup"
backups=( "2","9000" "3","9001" "4","9002")

client_cmd="$GOPATH/bin/vrgo --mode=client --port=1234 --id=123"

go install ./...

for element in ${backups[@]}; do
	IFS=',' read id port <<< "${element}"
	eval "${backup_cmd}  --port=${port} --id=${id}> $GOPATH/bin/backup${id}-${port}.log&"
	echo "Running ${backup_cmd}  --port=${port} --id=${id} > $GOPATH/bin/backup${id}-${port}.log&"
done

eval ${primary_cmd}
echo "Running ${primary_cmd}"
eval ${client_cmd}
