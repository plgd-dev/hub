# PLGD Dashboard

## Building

### Requirements

```bash
nodejs
```

### Configuration

Configuration for the client can be found in `./http-gateway/web/src/public/web_configuration.json`.

```json
{
  "domain": "auth.plgd.cloud",
  "httpGatewayAddress": "https://api.try.plgd.cloud",
  "webOAuthClient": {
    "clientID": "pHdCKhnpgGEtU7KAPcLYCoCAkZ4cYVZg",
    "audience": "https://try.plgd.cloud",
    "scopes": []
  },
  "deviceOAuthClient": {
    "clientID": "cYN3p6lwNcNlOvvUhz55KvDZLQbJeDr5",
    "audience": "",
    "scopes": ["offline_access"],
    "providerName": "plgd"
  }
}
```

### Installation

```bash
$ npm install
```

### Starting the development server

```bash
$ npm start
```

Application will be hosted on `http://localhost:3000` by default. To change the default port, put `PORT=xxxx` into `package.json` script for starting the development server

```bash
cross-env PORT=3000 craco start
```

or set `PORT` into your environment variables.

### Building the app

```bash
$ npm run build
```

## Translation

### Adding a new language

In order to add a new language, open the file `./http-gateway/web/src/config.js` and extend the `supportedLanguages` array with additional values. Use the [language code 2](https://www.science.co.il/language/Codes.php) for the language you are about to add:

`supportedLanguages: ['en', 'sk']`

Once you added a new language, open the file `./http-gateway/web/src/components/language-switcher/language-switcher-i18n.js` and add a new entry for the language you added. For example, for the language with code `sk` the entry would look like this:

```javascript
sk: {
  id: 'language-switcher.slovak',
  defaultMessage: 'Slovak',
},
```

### Generating language files

For extracting the messages from the UI components, run the following script:

```bash
$ npm run generate-pot
```

This script will generate a `template.pot` file, which contains all the strings from the application ready to be translated. Upload this file to your translation tool, translate the strings and after that, export the `.po` files for all the translations and place them into `./i18n` folder.

For generating language files which are used by the application, run the following script:

```bash
$ npm run generate-language-files
```

Now your translations are updated and ready to be used.

### Default language

The default language is set to be `en`. This configuration can be overridden in `./http-gateway/web/src/config.js` by changing the `defaultLanguage` field. Change this only to a language that is supported by the application (is present in the `supportedLanguages` list).

After your first visit to the application from a browser, the application will remember the current language state. In order to change the `defaultLanguage` field and see the change, you should clear the `localStorage` entry `language` from the browser.

## Branding

### Colors change

All colors are defined in one `scss` file: `./http-gateway/web/src/common/styles/colors.scss`. Changing one of the colors will have an impact on all parts of the application.

### Logo change

You can change the logo of the application by replacing these files:

Big logo (when the menu is expanded):

- `./http-gateway/web/src/assets/img/logo-big.svg`
  _Recommended size is 127px \* 35px_

Small logo (when the menu is collapsed):

- `./http-gateway/web/src/assets/img/logo-small.svg`
  _Recommended size is 45px \* 35px_

Favicon:

- `./http-gateway/web/public/favicon.png`

You might need to adjust some CSS in order to have the Logo rendered correctly, if the size is different than the recommended one. You can modify these values in `./http-gateway/web/src/components/left-panel/left-panel.scss`, look for the classes `.logo-big` and `.logo-small`. Adjust the height in these classes to fit your needs.

### Application name

The application name which also appears in the title bar can be changed by modifying the `appName` field in `./http-gateway/web/src/config.js`

### Header / Status Bar

The header has a dedicated component which can be found in `./http-gateway/web/src/components/status-bar/status-bar.js`. You can modify the status-bar `<header id="status-bar">...</header>` by removing existing components like `LanguageSwitcher` and `UserWidget` or by adding different content in the `header`.

### Footer

Footer has a dedicated component which can be found in `./http-gateway/web/src/components/footer/footer.js`. You can modify the `footer` tag by removing the already present links, or simply adding different content next to them.

### Text changes

Every text in this application is coming from a translation file located in `./http-gateway/web/src/languages/langauges.json`. This object contains a language block for each language you support in your application. If a block is missing you can duplicate an existing block and modify the block with the language code that is missing.

Some messages might be missing. This is due to fact that they were not yet translated. You can add them manually or use a language editor like [POEditor](https://poeditor.com/).

You can also override these strings to fit your need, for example, if you would like to have `Devices` as the name of the menu which we call `Things`, you have to find the key `menu.things` and change its value to `Devices`.
