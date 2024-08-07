import React, { FC, useContext, useState } from 'react'
import { useIntl } from 'react-intl'

import { useForm } from '@shared-ui/common/hooks'
import { FormContext } from '@shared-ui/common/context/FormContext'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'

import { messages as t } from '../../../EnrollmentGroups.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { DetailFromChunk2 } from '@/containers/DeviceProvisioning/EnrollmentGroups/DetailFormChunks'
import { Inputs } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.types'
import notificationId from '@/notificationId'
import { useValidationsSchema } from '@/containers/DeviceProvisioning/EnrollmentGroups/validationSchema'

const Step2: FC<any> = (props) => {
    const { defaultFormData } = props

    const { setStep } = useContext(FormContext)
    const { formatMessage: _ } = useIntl()
    const schema = useValidationsSchema('group2')

    const {
        formState: { errors, isValid },
        control,
        updateField,
        setValue,
        watch,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'step2',
        schema,
    })

    const [error, setError] = useState(false)

    return (
        <form>
            <FullPageWizard.Headline>{_(t.deviceAuthentication)}</FullPageWizard.Headline>
            <FullPageWizard.Description>{_(t.addEnrollmentGroupDeviceAuthenticationDescription)}</FullPageWizard.Description>

            <DetailFromChunk2
                control={control}
                errorNotificationId={notificationId.HUB_DPS_LINKED_HUBS_ADD_NEW_PAGE_CERT_PARSE_ERROR}
                errors={errors}
                setError={setError}
                setValue={setValue}
                updateField={updateField}
                watch={watch}
            />

            <StepButtons
                disableNext={!isValid || error}
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
