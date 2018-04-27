#!/bin/sh

gometalinter \
--deadline=90s \
--concurrency=6 \
--vendor \
--cyclo-over=15 \
--tests \
--exclude="exported (var)|(method)|(const)|(type)|(function) [A-Za-z\.0-9]* should have comment" \
--exclude="comment on exported function [A-Za-z\.0-9]* should be of the form" \
--exclude="Api.* should be .*API" \
--exclude="Http.* should be .*HTTP" \
--exclude="Id.* should be .*ID" \
--exclude="Json.* should be .*JSON" \
--exclude="Url.* should be .*URL" \
--exclude="Uuid.* should be .*UUID" \
--exclude="duplicate of" \
--exclude="can be fmt\.Stringer" \
--exclude="error return value not checked \(.*\.Audit" \
--exclude="error return value not checked \(.*\.Info" \
--exclude="error return value not checked \(.*\.Warning" \
./...
