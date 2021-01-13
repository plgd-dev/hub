/* eslint-disable */
'use strict';

const _ = require('shelljs');

const templateJsonFile = './i18n/template.json';
const outputLanguageJsonFile = './src/languages/languages.json';

// generate languages.json file from the mapped .po files.
_.exec(`rip po2json "./i18n/**/*.po" -m "${templateJsonFile}" -o "${outputLanguageJsonFile}"`);
