import fart1 from '@/assets/audio/fart1.mp3'
import fart2 from '@/assets/audio/fart2.mp3'
import fart3 from '@/assets/audio/fart3.mp3'

export const playRandomFartSound = () => {
  const sounds = [fart1, fart2, fart3]
  const random = Math.floor(Math.random() * sounds.length)
  const sound = new Audio(sounds[random])

  sound?.load?.()
  sound?.play?.()
}
