import React, { FC, useContext, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { useForm, WellKnownConfigType } from '@shared-ui/common/hooks'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import { FormContext } from '@shared-ui/common/context/FormContext'
import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import FormTextarea from '@shared-ui/components/Atomic/FormTextarea'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Button, { buttonVariants } from '@shared-ui/components/Atomic/Button'
import { security } from '@shared-ui/common/services'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { Props, Inputs } from './Step3.types'
import { messages as g } from '@/containers/Global.i18n'
import { useConfigurationList } from '@/containers/SnippetService/hooks'
import AddNewTokenModal from '@/containers/ApiTokens/AddNewTokenModal'
import { CreateTokenReturnType } from '@/containers/ApiTokens/ApiTokens.types'
import testId from '@/testId'

const Step3: FC<Props> = (props) => {
    const { defaultFormData, onFinish } = props

    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)
    const { data, loading: loadingProp } = useConfigurationList()

    const [loading, setLoading] = useState(false)
    const [addTokenModal, setAddTokenModal] = useState(false)

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const {
        formState: { errors },
        updateField,
        register,
        watch,
        setValue,
    } = useForm<Inputs>({
        defaultFormData,
        errorKey: 'tab3',
    })

    const options = useMemo(
        () =>
            loading || loadingProp
                ? []
                : data?.map((item: { name: string; id: string; version: string }) => ({
                      label: `${item.name} - v${item.version} - ${item.id}`,
                      value: item.id,
                  })) || [],
        [data, loading, loadingProp]
    )

    const configurationId = watch('configurationId')
    const apiAccessToken = watch('apiAccessToken')

    const handleUpdateToken = (data: CreateTokenReturnType) => {
        setValue('apiAccessToken', data.accessToken)
        updateField('apiAccessToken', data.accessToken)

        setLoading(false)
    }

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.selectConfiguration)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.selectConfigurationShortDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>{_(confT.selectConfigurationSubHeadline)}</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>{_(confT.selectConfigurationDescription)}</FullPageWizard.Description>

            <FormGroup error={errors.configurationId ? _(g.requiredField, { field: _(confT.configuration) }) : undefined} id='configurationId'>
                <FormLabel required text={_(confT.selectConfiguration)} />
                <FormSelect
                    isClearable
                    dataTestId={testId.snippetService.conditions.addPage.step3.selectConfiguration}
                    error={!!errors.configurationId}
                    onChange={(option: OptionType) => {
                        const v = option ? option.value : ''
                        setValue('configurationId', v.toString())
                        updateField('configurationId', v)
                    }}
                    options={options}
                    value={configurationId ? options?.find((o: OptionType) => o.value === configurationId) : null}
                />
            </FormGroup>

            <FormGroup error={errors.apiAccessToken ? _(g.requiredField, { field: _(g.name) }) : undefined} id='apiAccessToken'>
                <FormLabel required text={_(confT.APIAccessToken)} />
                <FormTextarea
                    dataTestId={testId.snippetService.conditions.addPage.step3.apiToken}
                    {...register('apiAccessToken', { required: true, validate: (val) => val !== '' })}
                    onBlur={(e) => updateField('apiAccessToken', e.target.value)}
                    style={{ height: 450 }}
                />
            </FormGroup>

            {wellKnownConfig?.m2mOauthClient?.clientId && (
                <>
                    <Spacer type='mt-3'>
                        <Button
                            dataTestId={testId.snippetService.conditions.addPage.step3.generateApiToken}
                            loading={loading}
                            loadingText={_(g.loading)}
                            onClick={(e) => {
                                e.preventDefault()
                                e.stopPropagation()
                                setAddTokenModal(true)
                            }}
                            variant={buttonVariants.SECONDARY}
                        >
                            {_(confT.generateNewToken)}
                        </Button>
                    </Spacer>
                    <AddNewTokenModal
                        dataTestId={testId.snippetService.conditions.addPage.step3.generateApiTokenModal}
                        defaultName={`${defaultFormData.name}-condition` || ''}
                        handleClose={() => setAddTokenModal(false)}
                        onSubmit={handleUpdateToken}
                        show={addTokenModal}
                    />
                </>
            )}

            <StepButtons
                dataTestId={testId.snippetService.conditions.addPage.step3.buttons}
                disableNext={!configurationId || !apiAccessToken || Object.keys(errors).length > 0}
                i18n={{
                    back: _(g.back),
                    continue: _(g.createAndSave),
                    formError: _(g.invalidFormState),
                    requiredMessage: _(g.requiredMessage),
                }}
                onClickBack={() => setStep?.(1)}
                onClickNext={onFinish}
            />
        </form>
    )
}

Step3.displayName = 'Step3'

export default Step3
