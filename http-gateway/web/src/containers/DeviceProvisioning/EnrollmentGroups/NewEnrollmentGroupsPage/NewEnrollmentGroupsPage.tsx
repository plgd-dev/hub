import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import cloneDeep from 'lodash/cloneDeep'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import usePersistentState from '@shared-ui/common/hooks/usePersistentState'
import { FormContext, getFormContextDefault } from '@shared-ui/common/context/FormContext'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { security } from '@shared-ui/common/services'
import { WellKnownConfigType } from '@shared-ui/common/hooks'
import { getOwnerId } from '@shared-ui/common/services/api-utils'

import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { DEFAULT_FORM_DATA } from '@/containers/DeviceProvisioning/EnrollmentGroups/NewEnrollmentGroupsPage/constants'
import { createEnrollmentGroup } from '@/containers/DeviceProvisioning/rest'
import { stringToPem } from '@/containers/DeviceProvisioning/utils'
import { pages } from '@/routes'
import notificationId from '@/notificationId'

const Step1 = lazy(() => import('./Steps/Step1'))
const Step2 = lazy(() => import('./Steps/Step2'))
const Step3 = lazy(() => import('./Steps/Step3'))

const NewEnrollmentGroupsPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const navigate = useNavigate()
    const { step } = useParams()

    const steps = useMemo(
        () => [
            {
                name: _(t.enrollmentConfiguration),
                description: _(t.tab1Description),
                link: pages.DPS.ENROLLMENT_GROUPS.NEW.TABS[0],
            },
            {
                name: _(t.deviceAuthentication),
                description: _(t.tab2Description),
                link: pages.DPS.ENROLLMENT_GROUPS.NEW.TABS[1],
            },
            {
                name: _(t.deviceCredentials),
                description: _(t.tab3Description),
                link: pages.DPS.ENROLLMENT_GROUPS.NEW.TABS[2],
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const defaultFormData = {
        ...DEFAULT_FORM_DATA,
        owner: getOwnerId(wellKnownConfig.jwtOwnerClaim || ''),
    }

    const [activeItem, setActiveItem] = useState(step ? steps.findIndex((s) => s.link.includes(step)) : 0)
    const [formData, setFormData, rehydrated] = usePersistentState<any>('dps-create-enrollment-group-form', defaultFormData)

    const onStepChange = useCallback(
        (item: number) => {
            setActiveItem(item)

            navigate(generatePath(pages.DPS.ENROLLMENT_GROUPS.NEW.LINK, { step: steps[item].link }))
        },
        [navigate, steps]
    )

    const onSubmit = async () => {
        const dataForSave = cloneDeep(formData)

        if (dataForSave.preSharedKey && dataForSave.preSharedKey !== '') {
            dataForSave.preSharedKey = stringToPem(dataForSave.preSharedKey)
        }

        await createEnrollmentGroup(dataForSave)

        setFormData(DEFAULT_FORM_DATA)

        Notification.success(
            { title: _(t.enrollmentGroupCreated), message: _(t.enrollmentGroupCreatedMessage) },
            { notificationId: notificationId.HUB_DPS_ENROLLMENT_GROUP_LIST_PAGE_CREATED }
        )

        navigate(generatePath(pages.DPS.ENROLLMENT_GROUPS.LINK))
    }

    const context = useMemo(
        () => ({
            ...getFormContextDefault(_(g.default)),
            updateData: (newFormData: any) => setFormData(newFormData),
            setFormError: () => {},
            setStep: onStepChange,
            onSubmit,
        }),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    return (
        <FullPageWizard
            activeStep={activeItem}
            i18n={{
                close: _(g.close),
            }}
            onClose={() => {
                setFormData(defaultFormData)
                navigate(generatePath(pages.DPS.ENROLLMENT_GROUPS.LINK))
            }}
            onStepChange={onStepChange}
            steps={steps}
            title={_(t.addEnrollmentGroup)}
        >
            <Loadable condition={rehydrated}>
                <FormContext.Provider value={context}>
                    <ContentSwitch activeItem={activeItem} style={{ width: '100%' }}>
                        <Step1 defaultFormData={formData} />
                        <Step2 defaultFormData={formData} />
                        <Step3 defaultFormData={formData} onSubmit={onSubmit} />
                    </ContentSwitch>
                </FormContext.Provider>
            </Loadable>
        </FullPageWizard>
    )
}

NewEnrollmentGroupsPage.displayName = 'NewEnrollmentGroupsPage'

export default NewEnrollmentGroupsPage
