#!/bin/bash

./php-scrambler

cd $POLYSCRIPT_PATH
s_php tok-php-transformer.php -p "$PHP_SRC_PATH"/ext/phar --replace --inc

cd $PHP_SRC_PATH
make install -k

cd $POLYSCRIPT_PATH
s_php tok-php-transformer.php -p "$PHP_SRC_PATH"/ext/phar/phar.php --replace

cd $PHP_SRC_PATH
make install -k
