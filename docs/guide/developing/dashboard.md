# PLGD Dashboard

## Building

### Requirements

```bash
nodejs
```

### Configuration

Configuration for the client can be found in `./src/auth_config.json`.

```json
{
  "domain": "",
  "clientID": "",
  "audience": "",
  "scope": "",
  "httpGatewayAddress": ""
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

## Branding

### Colors change

All colors are defined in one `scss` file located in `/src/common/styles/colors.scss`. Changing one of the colors will have an impact on all parts of the application.

### Logo change

You can change the logo of the application by replacing these files:

Big logo (when the menu is expanded):

- `/src/assets/img/logo-big.svg`
  _Recommended size is 127px \* 35px_

Small logo (when the menu is collapsed):

- `/src/assets/img/logo-small.svg`
  _Recommended size is 45px \* 35px_

Favicon:

- `/public/favicon.png`
