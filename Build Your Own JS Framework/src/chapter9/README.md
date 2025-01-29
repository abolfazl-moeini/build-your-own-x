# Chapter 9

## Interactive script

In this directory run `npm install` and then `npm start` to quickly get started with the examples from this chapter.

## Contents

* `commitizen` consists of a project uses the [Commitizen](https://github.com/commitizen/cz-cli) package for enforcing git commit template.
* `ci` directory consists of an example project that showcases the continuous integration maintenance using Circle CI and GitHub Actions. It shows you the minimal configuration to enable the CI flows in your project.
* `maintenance-tools` showcases several Node.js utilities used for framework maintenance. The following scripts are available:
  * `npm run release` - runs the Release It! (https://github.com/release-it/release-it) script as a demonstration in the current project. 
  * `npm run check-updates` - runs the Check Updates (https://github.com/raineorshine/npm-check-updates) script to list the dependencies that can be updated in the project.
  * `npm run license` - generates a report (https://github.com/ironSource/license-report) of dependency licenses, relevant to the licensing part of the chapter.