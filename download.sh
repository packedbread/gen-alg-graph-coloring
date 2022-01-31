#!/bin/bash
mkdir dataset && mkdir dataset/data
wget -O dataset.tar https://mat.tepper.cmu.edu/COLOR/instances/instances.tar
tar -xf dataset.tar -C dataset/data && rm dataset.tar

mkdir dataset/converter
wget -O dataset/converter.shar https://mat.tepper.cmu.edu/COLOR/format/binformat.shar
chmod +x dataset/converter.shar
(cd dataset/converter && ../converter.shar && make) && rm dataset/converter.shar
