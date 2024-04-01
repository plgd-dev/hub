import React, { FC, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Control, Controller, FieldErrors, UseFormRegister, UseFormSetValue } from 'react-hook-form'
import get from 'lodash/get'

import Dropzone from '@shared-ui/components/Atomic/Dropzone'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import CaList from '@shared-ui/components/Organisms/CaList/CaList'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import Switch from '@shared-ui/components/Atomic/Switch'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import { formatCertName } from '@shared-ui/common/services/certificates'
import { useCaData } from '@shared-ui/common/hooks'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import ShowAnimate from '@shared-ui/components/Atomic/ShowAnimate'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'

import { messages as t } from '@/containers/DeviceProvisioning/EnrollmentGroups/EnrollmentGroups.i18n'
import { nameLengthValidator, stringToPem } from '@/containers/DeviceProvisioning/utils'
import { messages as g } from '@/containers/Global.i18n'
import { useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { Inputs } from './EnrollmentGroups.types'

type Chunk2Props = {
    certificateChain?: string
    control: Control<Inputs, any>
    errorNotificationId?: string
    errors: FieldErrors<Inputs>
    isEditMode?: boolean
    setValue: UseFormSetValue<Inputs>
    updateField: (field: any, fieldValue: any) => void
}

export const DetailFromChunk2: FC<Chunk2Props> = (props) => {
    const { certificateChain, control, isEditMode, setValue, updateField, errors, errorNotificationId } = props

    const { formatMessage: _ } = useIntl()
    const i18nCert = useCaI18n()

    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: '',
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const { parsedData: certData } = useCaData({
        data: certificateChain && certificateChain !== '' ? [certificateChain] : [],
        onError: (e) => Notification.error({ title: _(t.certificationParsingError), message: e }, { notificationId: errorNotificationId }),
    })

    const handleViewCa = useCallback(
        (id: number) => {
            const caItem = certData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.certificateDetail), subTitle: caItem.name, data: caItem.data || caItem.name, dataChain: caItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    const leadCertificates = useMemo(
        () =>
            certData && certData[0] && certData[0].data?.length > 1
                ? certData &&
                  certData[0] &&
                  certData[0].data.map((cert: any, key: number) => ({
                      value: formatCertName(cert, true),
                      label: formatCertName(cert),
                  }))
                : [],
        [certData]
    )

    const middleRows = [
        certificateChain && !certificateChain.startsWith('/') && certData && certData[0] && certData[0].data?.length > 1
            ? {
                  attribute: _(t.leadCertificate),
                  value: (
                      <FormGroup error={get(errors, 'attestationMechanism.x509.leadCertificateName.message')} id='matchingCertificate'>
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
                                              const v = options.value
                                              onChange(v)
                                              updateField('attestationMechanism.x509.leadCertificateName', v)
                                          }}
                                          options={leadCertificates}
                                          size='small'
                                          value={value ? leadCertificates.filter((v: { value: string }) => value === v.value) : []}
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
                                updateField('attestationMechanism.x509.expiredCertificateEnabled', e.target.checked)
                            }}
                            size='small'
                        />
                    )}
                />
            ),
        },
    ].filter((i) => !!i) as Row[]

    return (
        <>
            {!isEditMode && <FormLabel required text={_(g.certificate)} />}
            {!certificateChain || certificateChain === '' ? (
                <>
                    <Dropzone
                        smallPadding
                        customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                        description={_(t.uploadCertDescription)}
                        maxFiles={1}
                        onFilesDrop={(files) => {
                            setTimeout(() => {
                                setValue('attestationMechanism.x509.certificateChain', stringToPem(files[0]), {
                                    shouldDirty: true,
                                    shouldTouch: true,
                                })
                                updateField(`attestationMechanism.x509.certificateChain`, stringToPem(files[0]))
                            }, 100)
                        }}
                        renderThumbs={false}
                        title={_(t.uploadCertTitle)}
                        validator={(file) => nameLengthValidator(file)}
                    />
                </>
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
                                onView: certificateChain?.startsWith('/') ? undefined : handleViewCa,
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

            <SimpleStripTable leftColSize={6} rightColSize={6} rows={middleRows} />

            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18nCert}
                onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
                title={caModalData.title}
            />
        </>
    )
}

type Chunk3Props = {
    errors: FieldErrors<Inputs>
    isEditMode?: boolean
    preSharedKeySettings: boolean
    register: UseFormRegister<Inputs>
    setPreSharedKeySettings: (psk: boolean) => void
    setValue: UseFormSetValue<Inputs>
    updateField: (field: any, fieldValue: any) => void
}

export const DetailFromChunk3: FC<Chunk3Props> = (props) => {
    const { register, isEditMode, setValue, updateField, errors, preSharedKeySettings, setPreSharedKeySettings } = props

    const { formatMessage: _ } = useIntl()

    const bottomRows = useMemo(
        () => [
            {
                attribute: _(t.preSharedKey),
                value: (
                    <FormGroup error={get(errors, 'preSharedKey.message')} id='preSharedKey'>
                        <FormInput inlineStyle={isEditMode} {...register('preSharedKey')} onBlur={(e) => updateField('preSharedKey', e.target.value)} />
                    </FormGroup>
                ),
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [errors.preSharedKey, preSharedKeySettings, register, updateField]
    )

    return (
        <>
            <Switch
                checked={preSharedKeySettings}
                label={_(t.preSharedKeySettings)}
                onChange={(e) => {
                    if (e.target.checked === false) {
                        updateField('preSharedKey', '')
                        setValue('preSharedKey', '')
                    }
                    setPreSharedKeySettings(e.target.checked)
                }}
                size='small'
            />

            <ShowAnimate show={preSharedKeySettings}>
                <Spacer type='pt-3'>
                    <SimpleStripTable leftColSize={6} rightColSize={6} rows={bottomRows} />
                </Spacer>
            </ShowAnimate>
        </>
    )
}
