#!/bin/sh
#

cat | ssh -i id_rsa -p 2342 www-data@vex.realraum.at set

exit 0
