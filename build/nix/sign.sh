#!/usr/bin/env sh
set -e
#
#
# Sign RPM's and push /dist artifacts to GH Release Assets
#
#
#
echo "===> Create .rpmmacros to sign rpm's from Goreleaser"
echo "%_gpg_name ${GPG_MAIL}" >> ~/.rpmmacros
echo "%_signature gpg" >> ~/.rpmmacros
echo "%_gpg_path /root/.gnupg" >> ~/.rpmmacros
echo "%_gpgbin /usr/bin/gpg" >> ~/.rpmmacros
echo "%__gpg_sign_cmd   %{__gpg} gpg --no-verbose --no-armor --batch --pinentry-mode loopback --passphrase ${GPG_PASSPHRASE} --no-secmem-warning -u "%{_gpg_name}" -sbo %{__signature_filename} %{__plaintext_filename}" >> ~/.rpmmacros

echo "===> Importing GPG private key from GHA secrets..."
printf %s ${GPG_PRIVATE_KEY_BASE64} | base64 -d | gpg --batch --import -

echo "===> Importing GPG signature, needed from Goreleaser to verify signature"
gpg --export -a ${GPG_MAIL} > /tmp/RPM-GPG-KEY-${GPG_MAIL}
rpm --import /tmp/RPM-GPG-KEY-${GPG_MAIL}

cd dist
# TODO @cciutea - review this step
for rpm_file in $(find -regex ".*\.\(rpm\)");do
  echo "===> Signing $rpm_file"
  rpm --addsign $rpm_file
  echo "===> Sign verification $rpm_file"
  rpm -v --checksig $rpm_file
done