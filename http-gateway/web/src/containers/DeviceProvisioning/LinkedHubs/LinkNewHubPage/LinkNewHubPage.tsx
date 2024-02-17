import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'

import * as styles from './LinkNewHubPage.styles'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../LinkedHubs.i18n'

const Step1 = lazy(() => import('./Steps/Step1'))
const Step2 = lazy(() => import('./Steps/Step2'))

const LinkNewHubPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    const [activeItem, setActiveItem] = useState(0)
    const [formData, setFormData] = useState()
    const [formError, setFormError] = useState({
        step1: false,
        step2: false,
    })

    const context = useMemo(
        () => ({
            ...getFormContextDefault(_(g.default)),
            updateData: (newFormData: any) => setFormData(newFormData),
            setFormError,
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const presetData = useCallback((data: any) => {
        setFormData(data)
        setActiveItem(1)
    }, [])

    return (
        <PageLayout headerBorder={true} title={_(t.linkNewHub)} xPadding={false}>
            <div css={styles.formContent}>
                <FormContext.Provider value={context}>
                    <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                        <Step1 defaultFormData={{ hubName: '', endpoinit: '' }} presetData={presetData} />
                        <Step2 defaultFormData={formData} />
                    </ContentSwitch>
                </FormContext.Provider>
            </div>
        </PageLayout>
    )
}

LinkNewHubPage.displayName = 'LinkNewHubPage'

export default LinkNewHubPage
