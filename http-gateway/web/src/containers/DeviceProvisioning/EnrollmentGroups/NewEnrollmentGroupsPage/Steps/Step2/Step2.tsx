import React, { FC, useContext } from 'react'
import { useIntl } from 'react-intl'

import * as commonStyles from '@shared-ui/components/Templates/FullPageWizard/FullPageWizardCommon.styles'
import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'

import { messages as t } from '../../../EnrollmentGroups.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { DetailFromChunk2 } from '@/containers/DeviceProvisioning/EnrollmentGroups/DetailFormChunks'
import { Inputs } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.types'
import notificationId from '@/notificationId'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

const Step2: FC<any> = (props) => {
    const { defaultFormData } = props

    const { updateData, setFormDirty, setFormError, setStep } = useContext(FormContext)
    const { formatMessage: _ } = useIntl()
    const {
        formState: { errors },
        control,
        updateField,
        setValue,
        watch,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        setFormDirty,
        errorKey: 'step2',
    })

    const certificateChain = watch('attestationMechanism.x509.certificateChain')

    return (
        <form>
            <h1 css={commonStyles.headline}>{_(t.deviceAuthentication)}</h1>
            <FullPageWizard.Description>{_(t.addEnrollmentGroupDeviceAuthenticationDescription)}</FullPageWizard.Description>

            <DetailFromChunk2
                certificateChain={certificateChain}
                control={control}
                errorNotificationId={notificationId.HUB_DPS_LINKED_HUBS_ADD_NEW_PAGE_CERT_PARSE_ERROR}
                errors={errors}
                setValue={setValue}
                updateField={updateField}
            />

            <StepButtons
                i18n={{
                    back: _(g.back),
                    continue: _(g.continue),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(0)}
                onClickNext={() => setStep?.(2)}
            />
        </form>
    )
}

Step2.displayName = 'Step2'

export default Step2
