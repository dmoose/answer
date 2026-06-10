/*
 * Network directory: profile extension, project list, tag catalog, member
 * search. Backend lives under /answer/api/v1/network/* (multisite build).
 */

import qs from 'qs';
import useSWR from 'swr';

import request from '@/utils/request';

export interface ProfileTag {
  id: string;
  slug: string;
  name: string;
  kind: number;
  description?: string;
}

export interface ProfileProject {
  id: string;
  title: string;
  description: string;
  repo_url: string;
  status: number;
  seeking_help: boolean;
  updated_at: number;
}

export interface ProfileExternalLink {
  label: string;
  url: string;
}

export interface NetworkProfile {
  user_id: string;
  display_name: string;
  avatar: string;
  global_rank: number;
  site_ranks?: Array<{
    site_id: string;
    site_name: string;
    site_slug: string;
    rank: number;
  }>;
  headline: string;
  pronouns: string;
  timezone: string;
  open_to_mentoring: boolean;
  open_to_collaboration: boolean;
  open_to_hire: boolean;
  external_links: ProfileExternalLink[];
  tags: ProfileTag[];
  projects: ProfileProject[];
}

export interface DirectoryMember {
  user_id: string;
  username: string;
  display_name: string;
  avatar: string;
  reputation: number;
  headline: string;
  pronouns: string;
  timezone: string;
  open_to_mentoring: boolean;
  open_to_collaboration: boolean;
  open_to_hire: boolean;
  tags: ProfileTag[];
}

export interface DirectorySearchParams {
  q?: string;
  tag_ids?: string[];
  open_to_mentoring?: boolean;
  open_to_collaboration?: boolean;
  open_to_hire?: boolean;
  page?: number;
  page_size?: number;
  sort?: 'rep_desc' | 'newest' | 'active';
}

export interface PageResult<T> {
  count: number;
  list: T[];
}

const cleanParams = <T extends Record<string, unknown>>(
  params: T,
): Partial<T> => {
  const out: Partial<T> = {};
  Object.keys(params).forEach((k) => {
    const v = params[k];
    if (v === undefined || v === null || v === '' || v === false) return;
    if (Array.isArray(v) && v.length === 0) return;
    out[k as keyof T] = v as T[keyof T];
  });
  return out;
};

export const getNetworkProfile = (userId: string) => {
  return request.get<NetworkProfile>(
    `/answer/api/v1/network/user/profile?user_id=${encodeURIComponent(userId)}`,
  );
};

export const useDirectorySearch = (params: DirectorySearchParams) => {
  const clean = cleanParams({
    ...params,
    page: params.page && params.page > 0 ? params.page : 1,
    page_size: params.page_size && params.page_size > 0 ? params.page_size : 20,
  });
  const qstr = qs.stringify(clean, { arrayFormat: 'repeat' });
  const url = `/answer/api/v1/network/members?${qstr}`;
  const { data, error, mutate } = useSWR<PageResult<DirectoryMember>>(
    url,
    request.instance.get,
  );
  return { data, error, isLoading: !data && !error, mutate };
};

export const useProfileTags = (kind?: number) => {
  const url = `/answer/api/v1/network/tags${kind ? `?kind=${kind}` : ''}`;
  const { data, error } = useSWR<ProfileTag[]>(url, request.instance.get);
  return { data, error, isLoading: !data && !error };
};

export const updateNetworkProfile = (params: {
  headline: string;
  pronouns: string;
  timezone: string;
  open_to_mentoring: boolean;
  open_to_collaboration: boolean;
  open_to_hire: boolean;
  external_links: ProfileExternalLink[];
}) => {
  return request.put('/answer/api/v1/network/profile', params);
};

export const setMyTags = (tagIds: string[]) => {
  return request.put('/answer/api/v1/network/me/tags', { tag_ids: tagIds });
};

export const createProject = (params: {
  title: string;
  description: string;
  repo_url: string;
  status: number;
  seeking_help: boolean;
}) => {
  return request.post<ProfileProject>(
    '/answer/api/v1/network/projects',
    params,
  );
};

export const updateProject = (
  id: string,
  params: {
    title: string;
    description: string;
    repo_url: string;
    status: number;
    seeking_help: boolean;
  },
) => {
  return request.put<ProfileProject>(
    `/answer/api/v1/network/projects/${id}`,
    params,
  );
};

export const deleteProject = (id: string) => {
  return request.delete(`/answer/api/v1/network/projects/${id}`);
};
