import fart from '@/assets/audio/fart.mp3'

const sound = new Audio(fart)

// We need to pre-load the sounds and play them from a click handler
// (this function is called on the first click on the page)
// so that browsers which are blocking autoplay from script
// will be able to play the notification sounds :)
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
