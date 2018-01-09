#!/bin/bash

USER=$NEXUS_USER
PASS=$NEXUS_PASS
echo "<settings>" > settings.xml
echo "<servers>" >> settings.xml
echo "<server>" >> settings.xml
echo "<id>nexus</id>" >> settings.xml
echo "<username>"$USER"</username>" >> settings.xml
echo "<password>"$PASS"</password>" >> settings.xml
echo "</server>" >> settings.xml
echo "</servers>" >> settings.xml
echo "</settings>" >> settings.xml

