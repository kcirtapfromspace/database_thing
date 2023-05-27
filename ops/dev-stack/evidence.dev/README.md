update node modules:

npm install -g npm-check-updates

ncu -u

npm update

npm install

Explanations:

To update all packages to new major versions, install the npm-check-updates package globally.

This will upgrade all the versions in the package.json file, of dependencies and devDependencies, so npm can install the new major versions.

You are now ready to run the update.

Now install updated packages.
The flag --force is sometimes required if there already exist some conflicting packages.