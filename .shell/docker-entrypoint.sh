#!/bin/sh
echo "$TZ" >  /etc/timezone

dir_root=/run/data
mkdir -p dir_root

if [ ! -L $dir_root/conf ]; then
dirs=('conf' 'qbot' 'scripts')
for d in ${dirs[*]}
do
rm -rf $dir_root/$d && mkdir -p /data/$d && ln -s /data/$d $dir_root/$d
done

f='.xdd.db'
rm -rm $dir_root/$f && touch $dir_root/$f &&  ln /data/$f $dir_root/$f
fi

exec /run/xdd