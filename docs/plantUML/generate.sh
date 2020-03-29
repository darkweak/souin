#!/bin/sh

for FILE in *.puml; do
  cat $FILE | docker run --rm -i think/plantuml > "${FILE%.puml}.svg"
done
echo Done
