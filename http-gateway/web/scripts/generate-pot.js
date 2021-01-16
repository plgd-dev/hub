/* eslint-disable */
'use strict';

const _ = require('shelljs');

const templateJsonFile = './i18n/template.json';
const outputTemplatePotFile = './i18n/template.pot';

// extract messages to template.json
_.exec(`formatjs extract "./src/**/*.js" --out-file "${templateJsonFile}" --format "./scripts/formatjs-extract-format.js"`);

// generate template.pot from template.json
_.exec(`rip json2pot "${templateJsonFile}" -o "${outputTemplatePotFile}"`);

// cleanup the template.json file
_.rm('-rf', templateJsonFile);
