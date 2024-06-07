import React, { FC } from 'react'
import { useIntl } from 'react-intl'

import FullPageWizard from '@shared-ui/components/Templates/FullPageWizard'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { useForm } from '@shared-ui/common/hooks'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import ResourceToggleCreator from '@shared-ui/components/Organisms/ResourceToggleCreator'

import { messages as confT } from '../../../../SnippetService.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Inputs } from '@/containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage/Steps/Step1/Step1.types'

const Step1: FC<any> = (props) => {
    const { defaultFormData } = props
    const { formatMessage: _ } = useIntl()

    const {
        formState: { errors },
        register,
        getValues,
        watch,
    } = useForm<Inputs>({ defaultFormData, errorKey: 'step1' })

    return (
        <form>
            <FullPageWizard.Headline>{_(confT.createConfig)}</FullPageWizard.Headline>
            <FullPageWizard.Description large>{_(confT.createConfigDescription)}</FullPageWizard.Description>

            <FullPageWizard.SubHeadline>Headline H4</FullPageWizard.SubHeadline>
            <FullPageWizard.Description>Popis čo tu uživateľ musí nastaviať a prípadne prečo</FullPageWizard.Description>

            <FullPageWizard.GroupHeadline>General</FullPageWizard.GroupHeadline>

            <FormGroup error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                <FormLabel text={_(g.name)} />
                <FormInput {...register('name', { required: true, validate: (val) => val !== '' })} />
            </FormGroup>

            <Spacer type='pt-15'>
                <FullPageWizard.GroupHeadline>{_(confT.listOfResources)}</FullPageWizard.GroupHeadline>

                <ResourceToggleCreator title='Title' />
            </Spacer>
        </form>
    )
}

Step1.displayName = 'Step1'

export default Step1
