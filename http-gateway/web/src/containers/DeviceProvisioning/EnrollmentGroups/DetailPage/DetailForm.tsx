import React, { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import { FormContext } from '@shared-ui/common/context/FormContext'
import { useCaData, useForm } from '@shared-ui/common/hooks'
import Switch from '@shared-ui/components/Atomic/Switch'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import CaList from '@shared-ui/components/Organisms/CaList/CaList'
import Dropzone from '@shared-ui/components/Atomic/Dropzone'
import Loadable from '@shared-ui/components/Atomic/Loadable/Loadable'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import { formatCertName } from '@shared-ui/common/services/certificates'
import ShowAnimate from '@shared-ui/components/Atomic/ShowAnimate/ShowAnimate'

import { Props, Inputs } from './DetailForm.types'
import { messages as g } from '../../../Global.i18n'
import { messages as t } from '../EnrollmentGroups.i18n'
import { useLinkedHubsList } from '@/containers/DeviceProvisioning/hooks'
import { nameLengthValidator, pemToString } from '@/containers/DeviceProvisioning/utils'
import { useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import notificationId from '@/notificationId'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { defaultFormData, resetIndex } = props

    const { updateData, setFormDirty, setFormError, commonInputProps, commonFormGroupProps } = useContext(FormContext)
    const { data: hubsData } = useLinkedHubsList()

    const {
        formState: { errors },
        register,
        control,
        updateField,
        reset,
        setValue,
        watch,
    } = useForm<Inputs>({
        defaultFormData,
        updateData,
        setFormError,
        setFormDirty,
        errorKey: 'tab1',
    })

    const i18n = useCaI18n()

    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: '',
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const [preSharedKeySettings, setPreSharedKeySettings] = useState(false)

    const linkedHubs = useMemo(
        () =>
            hubsData
                ? hubsData.map((linkedHub: { name: string; id: string }) => ({
                      value: linkedHub.id,
                      label: linkedHub.name,
                  }))
                : [],
        [hubsData]
    )

    useEffect(() => {
        if (resetIndex) {
            reset()
        }
    }, [reset, resetIndex])

    const topRows = useMemo(
        () => [
            {
                attribute: _(g.name),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.name ? _(g.requiredField, { field: _(g.name) }) : undefined} id='name'>
                        <FormInput
                            {...commonInputProps}
                            {...register('name', { required: true, validate: (val) => val !== '' })}
                            onBlur={(e) => updateField('name', e.target.value)}
                            placeholder={_(g.name)}
                        />
                    </FormGroup>
                ),
            },
            {
                attribute: _(t.linkedHubs),
                value: (
                    <FormGroup {...commonFormGroupProps} error={errors.hubIds ? _(g.requiredField, { field: _(t.linkedHubs) }) : undefined} id='linkedHubs'>
                        <div>
                            <Controller
                                control={control}
                                name='hubIds'
                                render={({ field: { onChange, value } }) => (
                                    <FormSelect
                                        inlineStyle
                                        isMulti
                                        align='right'
                                        error={!!errors.hubIds}
                                        onChange={(options: OptionType[]) => {
                                            const v = options.map((option) => option.value)
                                            onChange(v)
                                            updateField('hubIds', v)
                                        }}
                                        options={linkedHubs}
                                        size='small'
                                        value={value ? linkedHubs.filter((linkedHub: { value: string }) => value.includes(linkedHub.value)) : []}
                                    />
                                )}
                            />
                        </div>
                    </FormGroup>
                ),
            },
            { attribute: _(g.ownerID), value: defaultFormData?.owner },
        ],
        [commonFormGroupProps, commonInputProps, control, defaultFormData?.owner, errors.hubIds, errors.name, linkedHubs, register, updateField]
    )

    const certificateChain = watch('attestationMechanism.x509.certificateChain')

    const { parsedData: certData } = useCaData({
        data: certificateChain !== '' ? [certificateChain] : [],
        onError: (e) =>
            Notification.error(
                { title: _(t.certificationParsingError), message: e },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_PAGE_CERT_PARSE_ERROR }
            ),
    })

    const leadCertificates = useMemo(
        () =>
            certData && certData[0] && certData[0].data?.length > 1
                ? certData &&
                  certData[0] &&
                  certData[0].data.map((cert: any, key: number) => ({
                      value: key,
                      label: formatCertName(cert),
                  }))
                : [],
        [certData]
    )

    const handleViewCa = useCallback(
        (id: number) => {
            const caItem = certData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.certificateDetail), subTitle: caItem.name, data: caItem.data || caItem.name, dataChain: caItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    const middleRows = [
        !certificateChain.startsWith('/') && certData && certData[0] && certData[0].data?.length > 1
            ? {
                  attribute: _(t.matchingCertificate),
                  value: (
                      <FormGroup
                          {...commonFormGroupProps}
                          error={errors?.attestationMechanism?.x509?.leadCertificateName ? _(g.requiredField, { field: _(t.matchingCertificate) }) : undefined}
                          id='matchingCertificate'
                      >
                          <div>
                              <Controller
                                  control={control}
                                  name='attestationMechanism.x509.leadCertificateName'
                                  render={({ field: { onChange, value } }) => (
                                      <FormSelect
                                          inlineStyle
                                          align='right'
                                          error={!!errors.name}
                                          onChange={(options: OptionType) => {
                                              const v = options.label
                                              onChange(v)
                                              updateField('attestationMechanism.x509.leadCertificateName', v)
                                          }}
                                          options={leadCertificates}
                                          size='small'
                                          value={value ? leadCertificates.filter((v: { label: string }) => value === v.label) : []}
                                      />
                                  )}
                              />
                          </div>
                      </FormGroup>
                  ),
              }
            : undefined,
        {
            attribute: _(t.enableExpiredCertificates),
            value: (
                <Controller
                    control={control}
                    name='attestationMechanism.x509.expiredCertificateEnabled'
                    render={({ field: { onChange, value } }) => (
                        <Switch
                            checked={value}
                            onChange={(e) => {
                                onChange(e)
                                console.log('!', e.target.checked)
                                updateField('attestationMechanism.x509.expiredCertificateEnabled', e.target.checked)
                            }}
                            size='small'
                        />
                    )}
                />
            ),
        },
    ].filter((i) => !!i) as Row[]

    const bottomRows = useMemo(
        () => [
            {
                attribute: _(t.preSharedKey),
                value: (
                    <FormGroup
                        {...commonFormGroupProps}
                        error={errors.preSharedKey ? _(g.requiredField, { field: _(t.preSharedKey) }) : undefined}
                        id='preSharedKey'
                    >
                        <FormInput
                            {...commonInputProps}
                            {...register('preSharedKey', { required: true, validate: (val) => val !== '' })}
                            onBlur={(e) => updateField('preSharedKey', e.target.value)}
                            placeholder={_(t.preSharedKey)}
                        />
                    </FormGroup>
                ),
            },
        ],
        [commonFormGroupProps, commonInputProps, errors.preSharedKey, register, updateField]
    )

    return (
        <div>
            <form>
                <Spacer type='mb-4'>
                    <Headline type='h6'>{_(t.enrollmentConfiguration)}</Headline>
                </Spacer>
                <SimpleStripTable leftColSize={6} rightColSize={6} rows={topRows} />
                <Spacer type='mt-8 mb-4'>
                    <Headline type='h6'>{_(t.deviceAuthentication)}</Headline>
                </Spacer>

                {certificateChain === '' ? (
                    <Dropzone
                        smallPadding
                        customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                        description={_(t.uploadCertDescription)}
                        maxFiles={1}
                        onFilesDrop={(files) => {
                            setTimeout(() => {
                                setValue('attestationMechanism.x509.certificateChain', pemToString(files[0]), {
                                    shouldDirty: true,
                                    shouldTouch: true,
                                })
                            }, 100)
                        }}
                        renderThumbs={false}
                        title={_(t.uploadCertTitle)}
                        validator={(file) => nameLengthValidator(file)}
                    />
                ) : (
                    <Spacer type='pt-3'>
                        <Loadable condition={certData.length > 0}>
                            <CaList
                                largePadding
                                actions={{
                                    onDelete: () => {
                                        setValue(`attestationMechanism.x509.certificateChain`, '', { shouldDirty: true, shouldTouch: true })
                                        updateField(`attestationMechanism.x509.certificateChain`, '')
                                    },
                                    onView: certificateChain.startsWith('/') ? undefined : handleViewCa,
                                }}
                                data={[{ id: 0, name: certData && certData[0] ? certData[0].name : certificateChain, data: [], dataChain: '' }]}
                                formGroupProps={{
                                    marginBottom: false,
                                }}
                                i18n={{
                                    title: _(g.certificate),
                                    download: _(g.download),
                                    delete: _(g.delete),
                                    view: _(g.view),
                                }}
                            />
                        </Loadable>
                    </Spacer>
                )}

                <SimpleStripTable rows={middleRows} />

                <CaPoolModal
                    data={caModalData?.data}
                    dataChain={caModalData?.dataChain}
                    i18n={i18n}
                    onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                    show={caModalData?.data !== undefined}
                    subTitle={caModalData.subTitle}
                    title={caModalData.title}
                />

                <Spacer type='mt-8 mb-4'>
                    <Headline type='h6'>{_(t.deviceCredentials)}</Headline>
                </Spacer>

                <Switch
                    checked={preSharedKeySettings}
                    label={_(t.preSharedKeySettings)}
                    onChange={(e) => {
                        setPreSharedKeySettings(e.target.checked)
                    }}
                    size='small'
                />

                <ShowAnimate show={preSharedKeySettings}>
                    <Spacer type='pt-3'>
                        <SimpleStripTable rows={bottomRows} />
                    </Spacer>
                </ShowAnimate>
            </form>
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
