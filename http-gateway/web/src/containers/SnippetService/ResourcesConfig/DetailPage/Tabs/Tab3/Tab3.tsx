import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import Headline from '@shared-ui/components/Atomic/Headline'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'

const Tab3: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    return <Headline type='h5'>{_(confT.listOfDeviceAppliedConfigurations)}</Headline>
}

Tab3.displayName = 'Tab3'

export default Tab3
