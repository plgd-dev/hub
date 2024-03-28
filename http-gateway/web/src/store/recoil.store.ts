import { atom } from 'recoil'

export const dirtyFormState = atom({
    key: 'dirty-form-state',
    default: false,
})

export const promptBlockState = atom<{ link: string; id?: string } | undefined>({
    key: 'prompt-block-state',
    default: undefined,
})
