import { onMounted, onUnmounted, ref, type Ref } from 'vue'
import {
  getFeatureFlags,
  subscribeFeatureFlags,
  type FeatureFlags,
} from '../services/featuresApi'

/** Reactive mirror of server FEATURE_* kill switches. */
export function useFeatureFlags(): { features: Ref<FeatureFlags> } {
  const features = ref<FeatureFlags>(getFeatureFlags())
  let unsub: (() => void) | undefined

  onMounted(() => {
    features.value = getFeatureFlags()
    unsub = subscribeFeatureFlags((f) => {
      features.value = f
    })
  })

  onUnmounted(() => {
    unsub?.()
  })

  return { features }
}
