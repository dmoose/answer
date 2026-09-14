/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import { siteURL } from './siteUrl';

describe('siteURL', () => {
  const { origin } = window.location;

  it('default site lives at the origin', () => {
    expect(siteURL({ slug: 'default' })).toBe(`${origin}/`);
    expect(siteURL({ slug: 'default' }, '/questions/1')).toBe(
      `${origin}/questions/1`,
    );
  });

  it('sub-sites are path-prefixed', () => {
    expect(siteURL({ slug: 'golang' })).toBe(`${origin}/s/golang/`);
    expect(siteURL({ slug: 'golang' }, '/questions/1/x')).toBe(
      `${origin}/s/golang/questions/1/x`,
    );
    expect(siteURL({ slug: 'golang' }, 'tags')).toBe(`${origin}/s/golang/tags`);
  });

  it('an admin-set base_url wins and is used verbatim', () => {
    expect(
      siteURL({ slug: 'golang', base_url: 'https://go.example.com/' }, '/q/1'),
    ).toBe('https://go.example.com/q/1');
    expect(siteURL({ slug: 'golang', base_url: '  ' }, '/q/1')).toBe(
      `${origin}/s/golang/q/1`,
    );
  });
});
