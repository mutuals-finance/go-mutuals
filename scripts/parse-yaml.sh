#!/bin/bash

function parse_yaml {
   sed -ne "s|^\([[:space:]]*\):|\1|" -e "s|^\([[:space:]]*\)\([a-zA-Z0-9_]*\)[[:space:]]*:[[:space:]]*[\"']\(.*\)[\"'][[:space:]]*\$|\1$(echo @|tr @ '\034')\2$(echo @|tr @ '\034')\3|p" -e "s|^\([[:space:]]*\)\([a-zA-Z0-9_]*\)[[:space:]]*:[[:space:]]*\(.*\)[[:space:]]*\$|\1$(echo @|tr @ '\034')\2$(echo @|tr @ '\034')\3|p"  "$1" |
   awk -F "$(echo @|tr @ '\034')" '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("%s%s=\"%s\"\n",vn, $2, $3);
      }
   }'
}