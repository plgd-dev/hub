import fart1 from '@/assets/audio/fart1.mp3'
import fart2 from '@/assets/audio/fart2.mp3'
import fart3 from '@/assets/audio/fart3.mp3'

const sound1 = new Audio(fart1)
const sound2 = new Audio(fart2)
const sound3 = new Audio(fart3)

export const loadAllFartSounds = () => {
  const sounds = [sound1, sound2, sound3]

  sounds.forEach(sound => {
    sound.muted = true
    sound.play()
    sound.onended = () => {
      sound.muted = false
      sound.onended = () => {}
    }
  })
}

export const playRandomFartSound = () => {
  const sounds = [sound1, sound2, sound3]
  const random = Math.floor(Math.random() * sounds.length)
  const sound = sounds[random]

  sound?.play?.()
}
