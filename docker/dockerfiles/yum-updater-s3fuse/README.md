```
docker run --rm --name yum-updater-s3fuse:1.0 \
-e AWS_ACCESS_KEY_ID=MYKEY \
-e AWS_SECRET_ACCESS_KEY=MYSECRET \
-e AWS_BUCKET="nr-repo-apt" \
-e AWS_MOUNT_PREFIX=/mnt/repo \
-e REPO_YUM_UPDATE_METADATA_PATH \
--privileged \
jportasa/yum-updater-s3fuse:1.0 sleep 1000000
```

```
docker run -it -d --name yum-updater-s3fuse \
-e AWS_ACCESS_KEY_ID= \
-e AWS_SECRET_ACCESS_KEY= \
-e AWS_BUCKET="nr-repo-apt" \
-e AWS_MOUNT_PREFIX=/mnt/repo \
-e REPO_YUM_UPDATE_METADATA_PATH=/yum/el/8/x86_64 \
--privileged \
jportasa/yum-updater-s3fuse:1.0 sleep 1000000
```