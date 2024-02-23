import React, { FC } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import Button from '@shared-ui/components/Atomic/Button'

import * as commonStyles from '@/containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage/LinkNewHubPage.styles'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './SubStepButtons.types'

const SubStepButtons: FC<Props> = (props) => {
    const { onClickBack, onClickNext } = props
    const { formatMessage: _ } = useIntl()

    return (
        <Spacer css={commonStyles.buttons} type='mt-10'>
            <Button
                onClick={(e) => {
                    e.preventDefault()
                    isFunction(onClickBack) && onClickBack()
                }}
                size='big'
                variant='tertiary'
            >
                {_(g.back)}
            </Button>
            <Button
                fullWidth
                onClick={(e) => {
                    e.preventDefault()
                    isFunction(onClickNext) && onClickNext()
                }}
                size='big'
                variant='primary'
            >
                {_(g.continue)}
            </Button>
        </Spacer>
    )
}

SubStepButtons.displayName = 'SubStepButtons'

export default SubStepButtons
