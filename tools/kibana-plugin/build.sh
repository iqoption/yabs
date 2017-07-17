#!/bin/bash

rm -fr gl_crash_plugin.zip
mkdir -p kibana/gl_crash_plugin
cp -r public kibana/gl_crash_plugin
cp index.js kibana/gl_crash_plugin
cp package.json kibana/gl_crash_plugin

zip -r gl_crash_plugin.zip kibana
rm -fr kibana
