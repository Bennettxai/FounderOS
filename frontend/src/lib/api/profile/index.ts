export * from './types';
export * from './profile';

import * as profileApi from './profile';

export const api = {
  updateProfile: profileApi.updateProfile,
  uploadProfilePhoto: profileApi.uploadProfilePhoto,
  deleteProfilePhoto: profileApi.deleteProfilePhoto,
};

export default api;
