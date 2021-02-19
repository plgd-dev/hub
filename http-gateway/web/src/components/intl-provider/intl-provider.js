import { IntlProvider as ReactIntlProvider } from 'react-intl'
import { createContext } from 'react'

import languages from '@/languages/languages.json'
import { useLocalStorage } from '@/common/hooks'
import appConfig from '@/config'

export const LanguageContext = createContext()

export const IntlProvider = ({ children }) => {
  const [language, setLanguage] = useLocalStorage(
    'language',
    appConfig.defaultLanguage
  )
  const providerProps = {
    setLanguage,
    language,
  }

  return (
    <LanguageContext.Provider value={providerProps}>
      <ReactIntlProvider
        messages={languages[language]}
        locale={language}
        defaultLocale={appConfig.defaultLanguage}
        onError={err => {
          if (err.code === 'MISSING_TRANSLATION') {
            // console.warn('Missing translation', err.message)
            return
          }
          throw err
        }}
      >
        {children}
      </ReactIntlProvider>
    </LanguageContext.Provider>
  )
}
