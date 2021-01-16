/* eslint-disable */
'use strict';

const _ = require('shelljs');

const templateJsonFile = './i18n/template.json';
const outputLanguageJsonFile = './src/languages/languages.json';

// extract messages to template.json
_.exec(`formatjs extract "./src/**/*.js" --out-file "${templateJsonFile}" --format "./scripts/formatjs-extract-format.js"`);

// generate languages.json file from the mapped .po files.
_.exec(`rip po2json "./i18n/**/*.po" -m "${templateJsonFile}" -o "${outputLanguageJsonFile}"`);

// cleanup the template.json file
_.rm('-rf', templateJsonFile);