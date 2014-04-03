#!/bin/bash

export CASSDOCK_CONTAINERS=`docker ps | grep "zmar.*cass.*:" | sed -e "s/\s\+/ /g" | cut -d' ' -f1 | sed -e :a -e N -e 's/\n/ /' -e ta`
export CASS_IPS=`docker inspect $CASSDOCK_CONTAINERS | \
grep IPAddress | \
sed 's/"IPAddress": "/ /g' | \
sed 's/",//g' | \
sed 's/ //g' | \
sed -e :a -e N -e 's/\n/,/' -e ta`

echo $CASS_IPS
