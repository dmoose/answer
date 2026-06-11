import { create } from 'zustand';

interface FeaturesControl {
  directory_enabled: boolean;
  update: (params: { directory_enabled?: boolean }) => void;
  reset: () => void;
}

const featuresControlStore = create<FeaturesControl>((set) => ({
  directory_enabled: false,
  update: (params) =>
    set((state) => ({
      ...state,
      ...(params.directory_enabled !== undefined && {
        directory_enabled: params.directory_enabled,
      }),
    })),
  reset: () => set({ directory_enabled: false }),
}));

export default featuresControlStore;
