import { IntlProvider as ReactIntlProvider } from 'react-intl'
import { createContext } from 'react'

import languages from '@/languages/languages.json'
import { useLocalStorage } from '@/common/hooks'

const DEFAULT_LANGUAGE = 'en'

export const LanguageContext = createContext(DEFAULT_LANGUAGE)

export const IntlProvider = props => {
  const [language, setLanguage] = useLocalStorage('language', DEFAULT_LANGUAGE)
  const providerProps = {
    setLanguage,
    language,
  }

  return (
    <LanguageContext.Provider value={providerProps}>
      <ReactIntlProvider
        messages={languages[language]}
        locale={language}
        defaultLocale={DEFAULT_LANGUAGE}
        onError={err => {
          if (err.code === 'MISSING_TRANSLATION') {
            // console.warn('Missing translation', err.message)
            return
          }
          throw err
        }}
      >
        {props.children}
      </ReactIntlProvider>
    </LanguageContext.Provider>
  )
}
