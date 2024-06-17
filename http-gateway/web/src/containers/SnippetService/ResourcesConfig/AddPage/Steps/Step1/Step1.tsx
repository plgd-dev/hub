import React, { FC, useContext, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import isEqual from 'lodash/isEqual'
import isEmpty from 'lodash/isEmpty'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import ResourceToggleCreator from '@shared-ui/components/Organisms/ResourceToggleCreator'
import Button, { buttonSizes, buttonVariants } from '@shared-ui/components/Atomic/Button'
import StepButtons from '@shared-ui/components/Templates/FullPageWizard/StepButtons'
import { FormContext } from '@shared-ui/common/context/FormContext'

import { messages as confT } from '../../../../SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Inputs } from './Step1.types'
import { useValidationsSchema } from '../../validationSchema'

const Step1: FC<any> = (props) => {
    const { defaultFormData } = props

    const { formatMessage: _ } = useIntl()
    const { setStep } = useContext(FormContext)

    const schema = useValidationsSchema('tab1')

    const {
        formState: { errors },
        register,
        setValue,
        watch,
        updateField,
        trigger,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step1', schema })

    const defaultResourceData = useMemo(
        () => ({
            href: '',
            timeToLive: '0',
            content: {
                data: '',
                contentType: 'string',
                coapContentFormat: 0,
            },
        }),
        []
    )

    const resourceI18n = useMemo(
        () => ({
            add: _(g.add),
            addContent: _(confT.addContent),
            close: _(g.close),
            compactView: _(g.compactView),
            content: _(g.content),
            default: _(g.default),
            duration: _(g.duration),
            edit: _(g.edit),
            fullView: _(g.fullView),
            href: _(g.href),
            name: _(g.name),
            placeholder: _(g.placeholder),
            requiredField: (field: string) => _(g.requiredField, { field }),
            timeToLive: _(g.timeToLive),
            unit: _(g.unit),
            update: _(g.update),
        }),
        [_]
    )

    const resources = watch('resources')
    const name = watch('name')

    const hasInvalidResource = resources?.some((resource) => resource.href === '' || resource.timeToLive === '' || isEmpty(resource.content))

    useEffect(() => {
        const validationResult = schema.safeParse(defaultFormData)

        if (!validationResult.success) {
            if (defaultFormData.name) {
                trigger('name')
            }
        }
    }, [defaultFormData, schema, trigger])

    const hasError = useMemo(() => {
        const hrefs: string[] = []
        let er = false

        resources.forEach((resource, index) => {
            if (hrefs.includes(resource.href)) {
                er = true
            }
            hrefs.push(resource.href)
        })

        return er
    }, [resources])

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.createConfig)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.createConfigDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>Headline H4</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>Popis čo tu uživateľ musí nastaviať a prípadne prečo</FullPageWizard.Description>

            <FullPageWizard.GroupHeadline>{_(g.general)}</FullPageWizard.GroupHeadline>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                <FormLabel required text={_(g.name)} />
                <FormInput {...register('name')} onBlur={(e) => updateField('name', e.target.value)} />
            </FormGroup>

            <Spacer type='pt-15'>
                <FullPageWizard.GroupHeadline>{_(confT.listOfResources)}</FullPageWizard.GroupHeadline>

                {resources &&
                    resources.map((resource, key) => (
                        <Spacer key={key} type='mb-2'>
                            <ResourceToggleCreator
                                defaultOpen={isEqual(resource, defaultResourceData)}
                                i18n={resourceI18n}
                                onDeleted={() => {
                                    const newResources = resources.filter((_, index) => index !== key)
                                    setValue('resources', newResources)
                                    updateField('resources', newResources)
                                }}
                                onUpdate={(data) => {
                                    const newResources = resources.map((r, index) => (index === key ? data : r))
                                    updateField('resources', newResources)
                                    setValue('resources', newResources)
                                }}
                                resourceData={resource}
                                title={`Resource #${key}`}
                                updateField={updateField}
                            />
                        </Spacer>
                    ))}

                <Spacer type='pt-2'>
                    <Button
                        onClick={(e) => {
                            e.preventDefault()
                            setValue('resources', [...resources, defaultResourceData])
                            updateField('resources', [...resources, defaultResourceData])
                        }}
                        size={buttonSizes.SMALL}
                        variant={buttonVariants.FILTER}
                    >
                        {_(confT.addResource)}
                    </Button>
                </Spacer>

                <StepButtons
                    disableNext={hasInvalidResource || !name || hasError}
                    i18n={{
                        back: _(g.back),
                        continue: _(g.continue),
                        formError: _(g.invalidFormState),
                        requiredMessage: _(g.requiredMessage),
                    }}
                    onClickNext={() => setStep?.(1)}
                />
            </Spacer>
        </form>
    )
}

Step1.displayName = 'Step1'

export default Step1
