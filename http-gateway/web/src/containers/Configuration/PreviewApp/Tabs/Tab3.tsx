import React, { FC, lazy, Suspense } from 'react'

import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import * as styles from './Tab3.styles'

const components = [
    { render: lazy(() => import('@shared-ui/stories/Form/ActionButton.snip')), name: 'Action Button' },
    {
        render: lazy(() => import('@shared-ui/stories/Form/Button.snip')),
        name: 'Button',
        variants: [{}, { disabled: true }, { icon: <IconPlus /> }, { loading: true }],
    },
    { render: lazy(() => import('@shared-ui/stories/Form/Checkbox.snip')), name: 'Checkbox' },
    { render: lazy(() => import('@shared-ui/stories/Form/ColorPicker.snip')), name: 'ColorPicker' },
    { render: lazy(() => import('@shared-ui/stories/Form/DatePicker.snip')), name: 'DatePicker' },
    { render: lazy(() => import('@shared-ui/stories/Form/Dropzone.snip')), name: 'Dropzone' },
    { render: lazy(() => import('@shared-ui/stories/Form/FormGroup.snip')), name: 'FormGroup' },
    { render: lazy(() => import('@shared-ui/stories/Form/FormInput.snip')), name: 'FormInput' },
    { render: lazy(() => import('@shared-ui/stories/Form/FormSelect.snip')), name: 'FormSelect' },
    { render: lazy(() => import('@shared-ui/stories/Form/Radio.snip')), name: 'Radio' },
    {
        render: lazy(() => import('@shared-ui/stories/Form/SplitButton.snip')),
        name: 'SplitButton',
        variants: [{}, { disabled: true }, { icon: <IconPlus /> }, { loading: true }],
    },
    {
        render: lazy(() => import('@shared-ui/stories/Form/Switch.snip')),
        name: 'Switch',
        variants: [{}, { disabled: true }, { loading: true }],
    },
]

export const Tab3: FC<any> = () => (
    <div css={styles.example}>
        <Suspense fallback='...'>
            {components.map((component) => {
                const Render = component.render

                if (component.variants) {
                    return component.variants.map((variant) => (
                        <Spacer key={`${component.name}-${Object.keys(variant).join('-')}`} type='py-4'>
                            <>
                                <h3>
                                    {component.name} {Object.keys(variant).join(' | ')}
                                </h3>
                                <Render {...variant} />
                            </>
                        </Spacer>
                    ))
                } else {
                    return (
                        <Spacer key={component.name} type='py-4'>
                            <>
                                <h3>{component.name}</h3>
                                <Render />
                            </>
                        </Spacer>
                    )
                }
            })}
        </Suspense>
    </div>
)

export default Tab3
