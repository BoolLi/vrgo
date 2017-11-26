#!/bin/bash
primary_cmd="$GOPATH/bin/vrgo --mode=primary --port=1234 --id=0 --backup_ports=9000 --backup_ports=9001 --backup_ports=9002 --backup_ports=9003> $GOPATH/bin/primary.log&"

backup_cmd="$GOPATH/bin/vrgo --mode=backup"
backups=( "1","9000" "2","9001" "3","9002" "4","9003")

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
