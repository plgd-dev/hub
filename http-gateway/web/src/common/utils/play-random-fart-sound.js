import fart from '@/assets/audio/fart.mp3'

const sound = new Audio(fart)

export const loadFartSound = () => {
  sound.muted = true
  sound.play()
  sound.onended = () => {
    sound.muted = false
    sound.onended = () => {}
  }
}

export const playFartSound = () => {
  sound?.play?.()
}
