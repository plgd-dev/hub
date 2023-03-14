import { createContext } from 'react'
import { AppContextType } from './AppContext.types'

export const AppContext = createContext<AppContextType>({
    collapsed: false,
    footerExpanded: false,
})
