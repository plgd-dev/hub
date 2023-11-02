import { ReactElement } from 'react'

export type Props = {
    children: (clientData: any, wellKnownConfig: any, initializedByAnother: boolean) => ReactElement
}
