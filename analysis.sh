#!/bin/sh
#ruby analysis.rb old '' jsons/*.json | tee analysis/old.txt
#ruby analysis.rb old_repository 'repository' jsons/*.json | tee analysis/old_repository.txt
#ruby analysis.rb old_payload 'payload' jsons/*.json | tee analysis/old_payload.txt
#ruby analysis.rb old_payload_pages 'payload,pages,i:0' jsons/*.json | tee analysis/old_payload_pages.txt
#ruby analysis.rb old_payload_member 'payload,member' jsons/*.json | tee analysis/old_payload_member.txt
#ruby analysis.rb old_payload_comment 'payload,comment' jsons/*.json | tee analysis/old_payload_comment.txt
#ruby analysis.rb old_payload_comment_user 'payload,comment,user' jsons/*.json | tee analysis/old_payload_comment_user.txt
#ruby analysis.rb old_payload_release 'payload,release' jsons/*.json | tee analysis/old_payload_release.txt
#ruby analysis.rb old_payload_release_author 'payload,release,author' jsons/*.json | tee analysis/old_payload_release_author.txt
#ruby analysis.rb old_payload_release_assets 'payload,release,assets,i:0' jsons/*.json | tee analysis/old_payload_release_assets.txt
#ruby analysis.rb old_payload_release_assets_uploader 'payload,release,assets,i:0,uploader' jsons/*.json | tee analysis/old_payload_release_assets_uploader.txt
#ruby analysis.rb old_payload_repository 'payload,repository' jsons/*.json | tee analysis/old_payload_repository.txt
#ruby analysis.rb old_payload_repository_owner 'payload,repository,owner' jsons/*.json | tee analysis/old_payload_repository_owner.txt
#ruby analysis.rb old_payload_team 'payload,team' jsons/*.json | tee analysis/old_payload_team.txt
ruby analysis.rb old_payload_pull_request 'payload,pull_request' jsons/*.json | tee analysis/old_payload_pull_request.txt