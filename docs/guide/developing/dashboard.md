# PLGD Dashboard

## Building

### Requirements

```bash
nodejs
```

### Configuration

Configuration for the client can be found in `./http-gateway/web/src/auth_config.json`.

```json
{
  "domain": "auth.plgd.cloud",
  "clientID": "pHdCKhnpgGEtU7KAPcLYCoCAkZ4cYVZg",
  "audience": "https://try.plgd.cloud",
  "scope": "",
  "httpGatewayAddress": "https://api.try.plgd.cloud"
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

Default language is set to be `en`. This configuration can be overridden in `./http-gateway/web/src/config.js` by changing the `defaultLanguage` field. Change this only to a language which is supported by the application (is present in the `supportedLanguages` list).

After the first visit of the application from a browser, it will remember the current language state, so in order to see the `defaultLanguage` field to work, you should clear the `localStorage` entry `language` from the browser.

## Branding

### Colors change

All colors are defined in one `scss` file `./http-gateway/web/src/common/styles/colors.scss`. Changing one of the colors will have an impact on all parts of the application.

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

Change the application name which also appears as a part of the Title bar in the browser header in file `./http-gateway/web/src/config.js` by changing the `appName` field.

### Header / Status Bar

Header has a dedicated component which can be found in `./http-gateway/web/src/components/status-bar/status-bar.js`. You can modify the `div` whith `id` of `status-bar` by removing the already present component like `LanguageSwitcher` or `UserWidget`, or simply adding other content next to these components.

### Footer

Footer has a dedicated component which can be found in `./http-gateway/web/src/components/footer/footer.js`. You can modify the `footer` tag by removing the already present links, or simply adding other content next to them.

### Text changes

Every text in this application is comming from a translation file located in `./http-gateway/web/src/languages/langauges.json`. This object contains as many blocks, as many languages you support in your application. If some of the blocks are missing you can simply duplicate them and use the same language code you used when you added a new language.

Some messages might be missing. This is due to fact that they were not yet translated. You can do that manually or automatically with some language manager like [POEditor](https://poeditor.com/).

You can also override these strings to fit your need, for example if you would like to have `Devices` as a name of the menu which we call `Things`, you have to find the key `menu.things` and change its value to `Devices`.
